package callr

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"precisiondosing-api-go/internal/utils/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type callError struct {
	ErrorMsg string `json:"error_msg"`
	Timeout  bool   `json:"timeout"`
}

func (e *callError) Error() string {
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	return "R script error"
}

func newCallError(errorMsg string, timeout bool) *callError {
	return &callError{
		ErrorMsg: errorMsg,
		Timeout:  timeout,
	}
}

type platform interface {
	Setup(cmd *exec.Cmd) error    // setpgid or no-op
	Assign(cmd *exec.Cmd) error   // job-object or no-op
	Teardown(cmd *exec.Cmd) error // kill group or parent
}

func (c *CallR) run(ids CallRIDs, adjust bool, errorMsg string, maxExecutionTime time.Duration) ([]byte, *callError) {
	// 1) prepare cmd & pipes
	cmd, pipes, err := c.prepareCommand(ids.JobID, adjust, errorMsg)
	if err != nil {
		return nil, newCallError(err.Error(), false)
	}

	// 2) platform setup (Setpgid or noop)
	if err = plat.Setup(cmd); err != nil {
		return nil, newCallError("platform setup failed: "+err.Error(), false)
	}

	// 3) start
	ctx, cancel := context.WithTimeout(context.Background(), maxExecutionTime)
	defer cancel()

	if err = cmd.Start(); err != nil {
		return nil, newCallError("start failed: "+err.Error(), false)
	}

	// 4) platform assign (job-object on Windows, no-op on Unix)
	if err = plat.Assign(cmd); err != nil {
		return nil, newCallError("platform assign failed: "+err.Error(), false)
	}

	// 5) capture stdout/stderr
	var (
		stdoutBuf = &bytes.Buffer{}
		wg        sync.WaitGroup
		done      = make(chan error, 1)
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(stdoutBuf, pipes.stdout)
	}()

	go c.captureAndLogStderr(pipes.stderr, ids.OderID)
	go func() { done <- cmd.Wait() }()

	// 6) wait or timeout
	var timedOut bool
	select {
	case err = <-done:
		// finished normally
	case <-ctx.Done():
		timedOut = true
		terr := plat.Teardown(cmd) // kill process/group
		if terr != nil {
			c.logger.Error("Teardown failed", log.Err(terr))
		}
		err = errors.New("timeout")
	}
	wg.Wait() // drain stdout

	if err != nil {
		return nil, newCallError(err.Error(), timedOut)
	}
	return stdoutBuf.Bytes(), nil
}

type pipes struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (c *CallR) prepareCommand(jobID uint, adjust bool, errorMsg string) (*exec.Cmd, *pipes, error) {
	wd := filepath.Dir(c.adjustScriptPath)
	script := filepath.Base(c.adjustScriptPath)

	jobIDStr := strconv.FormatUint(uint64(jobID), 10)
	adjustStr := "FALSE"
	if adjust {
		adjustStr = "TRUE"
	}

	fullModelPath, err := filepath.Abs(c.modelPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get absolute path of model: %w", err)
	}

	//nolint:gosec // args are controlled and validated
	cmd := exec.Command(
		c.rscriptPath,
		script,
		jobIDStr,
		adjustStr,
		errorMsg,
		fullModelPath,
	)

	cmd.Dir = wd
	cmd.Env = append(os.Environ(),
		"R_MYSQL_PASSWORD="+c.mysqlPassword,
		"R_MYSQL_USER="+c.mysqlUser,
		"R_MYSQL_HOST="+c.mysqlHost,
		"R_MYSQL_DB="+c.mysqlDB,
		"R_MYSQL_TABLE=orders",
		"R_WORKER="+strconv.Itoa(c.rWorker),
	)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get stdout pipe: %w", err)
	}

	pipes := &pipes{
		stdout: stdoutPipe,
		stderr: stderrPipe,
	}

	return cmd, pipes, nil
}

func isExpectedPipeError(err error) bool {
	return strings.Contains(err.Error(), "file already closed") ||
		strings.Contains(err.Error(), "use of closed file") ||
		strings.Contains(err.Error(), "read/write on closed pipe")
}

func (c *CallR) captureOutput(stdout io.Reader, cmd *exec.Cmd, buf *bytes.Buffer, done chan<- error) {
	_, copyErr := io.Copy(buf, stdout)
	if copyErr != nil && !isExpectedPipeError(copyErr) {
		done <- copyErr
		return
	}
	done <- cmd.Wait()
}

func (c *CallR) captureAndLogStderr(stderr io.ReadCloser, orderID string) {
	scanner := bufio.NewScanner(stderr)

	for scanner.Scan() {
		line := scanner.Text()

		// this is a workaround for output of ospsuite that ignores R-rules for
		// suppression of warnings and errors
		if strings.HasPrefix(line, "[CVODES") || strings.HasPrefix(line, "\tInternal t") {
			continue
		}

		c.rLogger.Debug(line,
			log.Str("orderID", orderID))
	}

	if err := scanner.Err(); err != nil && !isExpectedPipeError(err) {
		c.logger.Error("failed to read stderr",
			log.Str("orderID", orderID),
			log.Err(err),
		)
	}
}

package callr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/utils/log"
	"strconv"
	"strings"
	"time"
)

type CallR struct {
	rscriptPath      string
	adjustScriptPath string
	mysqlPassword    string
	mysqlUser        string
	mysqlHost        string
	mysqlDB          string
	modelPath        string
	rWorker          int
	debugMode        bool
	logger           log.Logger
}

func New(
	rscriptPath string,
	adjustScriptPath string,
	dbConfig cfg.DatabaseConfig,
	modelPath string,
	rWorker int,
	debug bool,
) *CallR {
	return &CallR{
		rscriptPath:      rscriptPath,
		adjustScriptPath: adjustScriptPath,
		mysqlPassword:    dbConfig.Password,
		mysqlUser:        dbConfig.Username,
		mysqlHost:        dbConfig.Host,
		mysqlDB:          dbConfig.DBName,
		rWorker:          rWorker,
		debugMode:        debug,
		modelPath:        modelPath,

		logger: log.WithComponent("callr"),
	}
}

type Resp struct {
	DoseAdjusted bool     `json:"dose_adjusted"`
	Error        bool     `json:"error"`     // only if R fails -> stops with error
	ErrorMsg     string   `json:"error_msg"` // R error message from stop
	CallStack    []string `json:"call_stack"`
}

type RError struct {
	error     error
	CallStack []string
}

func (e *RError) Error() string {
	var parts []string
	parts = append(parts, "R script error")

	if len(e.CallStack) > 0 {
		parts = append(parts, "with call stack")
	}

	if e.error != nil {
		parts = append(parts, e.error.Error())
	}

	return strings.Join(parts, ": ")
}

func (e *RError) Unwrap() error {
	return e.error
}

func newRError(err error, callStack []string) *RError {
	return &RError{
		error:     err,
		CallStack: callStack,
	}
}

// error is always a non-recoverable system error
func (c *CallR) Adjust(jobID uint, adjust bool, errorMsg string, maxExecutionTime time.Duration) (*Resp, *RError) {
	bytes, err := c.call(jobID, adjust, errorMsg, maxExecutionTime)
	if err != nil {
		if err.Timeout {
			// timeout error -> we will retry one time with and error message job
			c.logger.Warn("script timed out", log.Str("jobID", strconv.FormatUint(uint64(jobID), 10)))

			msg := "The adjustment timed out (took too long)"
			retryBytes, retryErr := c.call(jobID, false, msg, maxExecutionTime)
			if retryErr != nil {
				return nil, newRError(retryErr, nil)
			}
			bytes = retryBytes
		} else {
			// real system error
			return nil, newRError(err, nil)
		}
	}

	var resp Resp
	if marshalErr := json.Unmarshal(bytes, &resp); marshalErr != nil {
		return nil, newRError(marshalErr, nil)
	}

	// Error in response -> R script stopped with error -> system error
	if resp.Error {
		return nil, newRError(errors.New(resp.ErrorMsg), resp.CallStack)
	}

	return &resp, nil
}

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

func (c *CallR) call(jobID uint, adjust bool, errorMsg string, maxExecutionTime time.Duration) ([]byte, *callError) {
	cmd, pipes, err := c.prepareCommand(jobID, adjust, errorMsg)
	if err != nil {
		return nil, newCallError(err.Error(), false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), maxExecutionTime)
	defer cancel()

	if err = cmd.Start(); err != nil {
		return nil, newCallError("failed to start script: "+err.Error(), false)
	}

	if err = assignProcessToJobObject(cmd); err != nil {
		c.logger.Warn("could not assign process to job object", log.Str("error", err.Error()))
	}

	var outputBuf bytes.Buffer
	done := make(chan error, 1)

	go c.captureOutput(pipes.stdout, cmd, &outputBuf, done)
	go c.captureAndLogStderr(pipes.stderr, jobID)

	select {
	case <-ctx.Done():
		killProcessGroup(cmd)
		return nil, newCallError("script timeout", true)
	case err = <-done:
		if err != nil {
			return nil, newCallError("script failed: "+err.Error(), false)
		}
	}

	return outputBuf.Bytes(), nil
}

func (c *CallR) captureAndLogStderr(stderr io.ReadCloser, jobID uint) {
	scanner := bufio.NewScanner(stderr)
	jobIDStr := strconv.FormatUint(uint64(jobID), 10)

	for scanner.Scan() {
		line := scanner.Text()
		c.logger.Debug("R output",
			log.Str("jobID", jobIDStr),
			log.Str("msg", line),
		)
	}

	if err := scanner.Err(); err != nil {
		c.logger.Error("stderr read error",
			log.Str("jobID", jobIDStr),
			log.Err(err),
		)
	}
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

	setCmdSysProcAttr(cmd)

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

func (c *CallR) captureOutput(stdout io.Reader, cmd *exec.Cmd, buf *bytes.Buffer, done chan<- error) {
	_, copyErr := io.Copy(buf, stdout)
	if copyErr != nil {
		done <- copyErr
		return
	}
	done <- cmd.Wait()
}

package callr

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
	rWorker          int
	logger           log.Logger
}

func New(
	rscriptPath string,
	adjustScriptPath string,
	mysqlHost string,
	mysqlDB string,
	mysqlUser string,
	mysqlPassword string,
	rWorker int,
) *CallR {
	return &CallR{
		rscriptPath:      rscriptPath,
		adjustScriptPath: adjustScriptPath,
		mysqlPassword:    mysqlPassword,
		mysqlUser:        mysqlUser,
		mysqlHost:        mysqlHost,
		mysqlDB:          mysqlDB,
		rWorker:          rWorker,
		logger:           log.WithComponent("callr"),
	}
}

type Resp struct {
	DoseAdjusted bool     `json:"dose_adjusted"`
	Error        bool     `json:"error"`     // only if R fails -> stops with error
	ErrorMsg     string   `json:"error_msg"` // R error message from stop
	CallStack    []string `json:"call_stack"`
	ProcessLog   string   `json:"process_log"`
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

	// log process from R script
	if resp.ProcessLog != "" {
		c.logger.Info("script process log",
			log.Str("jobID", strconv.FormatUint(uint64(jobID), 10)),
			log.Str("process_log", resp.ProcessLog),
		)
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
	wd := filepath.Dir(c.adjustScriptPath)
	script := filepath.Base(c.adjustScriptPath)

	jobIDStr := strconv.FormatUint(uint64(jobID), 10)
	adjustStr := "FALSE"
	if adjust {
		adjustStr = "TRUE"
	}

	ctx, cancel := context.WithTimeout(context.Background(), maxExecutionTime)
	defer cancel()

	// Rscript script.R jobID Adjust ErrorMsg
	// Rscript script.R int Bool(TRUE/FALSE) String
	cmd := exec.CommandContext(ctx, c.rscriptPath, script, jobIDStr, adjustStr, errorMsg) //nolint:gosec // no problem here
	cmd.Dir = wd
	cmd.Env = append(os.Environ(),
		"R_MYSQL_PASSWORD="+c.mysqlPassword,
		"R_MYSQL_USER="+c.mysqlUser,
		"R_MYSQL_HOST="+c.mysqlHost,
		"R_MYSQL_DB="+c.mysqlDB,
		"R_MYSQL_TABLE=orders",
		"R_WORKER="+strconv.Itoa(c.rWorker),
	)

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, newCallError("script timeout", true)
		}

		return nil, newCallError(err.Error(), false)
	}

	return out, nil
}

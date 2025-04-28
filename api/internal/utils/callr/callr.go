package callr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"precisiondosing-api-go/internal/utils/log"
	"strconv"
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
	DoseAdjusted bool   `json:"dose_adjusted"`
	Error        bool   `json:"error"`     // only if R fails -> stops with error
	ErrorMsg     string `json:"error_msg"` // R error message from stop
	ProcessLog   string `json:"process_log"`
}

// error is always a non-recoverable system error
func (c *CallR) Adjust(jobID uint, adjust bool, errorMsg string, maxExecutionTime time.Duration) (*Resp, error) {
	bytes, err := c.call(jobID, adjust, errorMsg, maxExecutionTime)
	if err != nil {
		if err.Timeout {
			// timeout error -> we will retry one time with and error message job
			c.logger.Warn("script timed out", log.Str("jobID", strconv.FormatUint(uint64(jobID), 10)))

			msg := "The adjustment timed out (took too long)"
			retryBytes, retryErr := c.call(jobID, false, msg, maxExecutionTime)
			if retryErr != nil {
				return nil, fmt.Errorf("retry script: %s", retryErr.ErrorMsg)
			}
			bytes = retryBytes
		} else {
			// real system error
			return nil, fmt.Errorf("script: %s", err.ErrorMsg)
		}
	}

	var resp Resp
	fmt.Println(string(bytes))
	if marshalErr := json.Unmarshal(bytes, &resp); marshalErr != nil {
		return nil, fmt.Errorf("cannot unmarshal script output: %w", marshalErr)
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
		return nil, fmt.Errorf("script error: %s", resp.ErrorMsg)
	}

	return &resp, nil
}

type Error struct {
	ErrorMsg string `json:"error_msg"`
	Timeout  bool   `json:"timeout"`
}

func (c *CallR) call(jobID uint, adjust bool, errorMsg string, maxExecutionTime time.Duration) ([]byte, *Error) {
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
			return nil, &Error{
				ErrorMsg: "script timeout",
				Timeout:  true,
			}
		}

		return nil, &Error{
			ErrorMsg: err.Error(),
			Timeout:  false,
		}
	}

	return out, nil
}

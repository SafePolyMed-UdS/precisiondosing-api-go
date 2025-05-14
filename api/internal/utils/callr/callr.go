package callr

import (
	"encoding/json"
	"errors"
	"github.com/cloudflare/ahocorasick"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/utils/log"
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
	rLogger          log.Logger
	rLogMatcher      *ahocorasick.Matcher
}

func New(
	rscriptPath string,
	adjustScriptPath string,
	dbConfig cfg.DatabaseConfig,
	modelPath string,
	rWorker int,
	debug bool,
) *CallR {
	rLogYellowList := []string{
		"[CVODE",
		"Internal t",
		"mxhnil",
	}
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

		logger:      log.WithComponent("callr"),
		rLogger:     log.WithComponent("rscript"),
		rLogMatcher: ahocorasick.NewStringMatcher(rLogYellowList),
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

type CallRIDs struct {
	JobID  uint
	OderID string
}

// error is always a non-recoverable system error
func (c *CallR) Adjust(ids CallRIDs, adjust bool, errorMsg string, maxExecutionTime time.Duration) (*Resp, *RError) {
	bytes, err := c.run(ids, adjust, errorMsg, maxExecutionTime)
	if err != nil {
		if err.Timeout {
			// timeout error -> we will retry one time with and error message job
			c.logger.Warn("script timed out", log.Str("OrderID", ids.OderID))

			errorMsg := "The adjustment timed out (took too long)"
			retryBytes, retryErr := c.run(ids, false, errorMsg, maxExecutionTime)
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

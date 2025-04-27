package callr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	}
}

// 1. Success -> R added stuff to the Db
// 2. Error in R -> R created Error PDF
// 3. Error in calling R -> R did nothing
// 4. Error in R -> R could not create Error PDF
type Resp struct {
	Success    bool   `json:"success"`
	ErrorMsg   string `json:"error_msg"`
	ProcessLog string `json:"process_log"`
}

func (c *CallR) Adjust(jobID uint, maxExecutionTime time.Duration) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), maxExecutionTime)
	defer cancel()

	wd := filepath.Dir(c.adjustScriptPath)
	script := filepath.Base(c.adjustScriptPath)

	// Rscript script.R jobID
	jobIDStr := strconv.FormatUint(uint64(jobID), 10)
	cmd := exec.CommandContext(ctx, c.rscriptPath, script, jobIDStr) //nolint:gosec // no problem here
	cmd.Dir = wd
	cmd.Env = append(os.Environ(),
		"R_MYSQL_PASSWORD="+c.mysqlPassword,
		"R_MYSQL_USER="+c.mysqlUser,
		"R_MYSQL_HOST="+c.mysqlHost,
		"R_MYSQL_DB="+c.mysqlDB,
		"R_MYSQL_TABLE=orders",
		"R_WORKER="+strconv.Itoa(c.rWorker),
	)

	// run
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("script timeout: %w", err)
		}
	}

	var resp Resp
	if err = json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("cannot unmarshal script output: %w", err)
	}

	return &resp, nil
}

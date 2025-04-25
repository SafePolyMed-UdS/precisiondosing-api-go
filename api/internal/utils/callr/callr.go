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
	maxExecutionTime time.Duration
}

func NewCallR(rscriptPath,
	adjustScriptPath,
	mysqlPassword,
	mysqlUser,
	mysqlHost,
	mysqlDB string,
	maxExecutionTime time.Duration,
) *CallR {
	return &CallR{
		rscriptPath:      rscriptPath,
		adjustScriptPath: adjustScriptPath,
		mysqlPassword:    mysqlPassword,
		mysqlUser:        mysqlUser,
		mysqlHost:        mysqlHost,
		mysqlDB:          mysqlDB,
		maxExecutionTime: maxExecutionTime,
	}
}

// 1. Success -> R added stuff to the Db
// 2. Error in R -> R created Error PDF
// 3. Error in calling R -> R did nothing
// 4. Error in R -> R could not create Error PDF
type Resp struct {
	Success    bool   `json:"success"`
	CreatedPDF bool   `json:"created_pdf"`
	MsgUser    string `json:"msg_user"`
	MsgSystem  string `json:"msg_system"`
}

func (c *CallR) Adjust(jobID int64) Resp {

	ctx, cancel := context.WithTimeout(context.Background(), c.maxExecutionTime)
	defer cancel()

	wd := filepath.Dir(c.adjustScriptPath)
	script := filepath.Base(c.adjustScriptPath)

	// Rscript script.R jobID
	cmd := exec.CommandContext(ctx, c.rscriptPath, script, strconv.FormatInt(jobID, 10))
	cmd.Dir = wd
	cmd.Env = append(os.Environ(),
		"R_MYSQL_PASSWORD="+c.mysqlPassword,
		"R_MYSQL_USER="+c.mysqlUser,
		"R_MYSQL_HOST="+c.mysqlHost,
		"R_MYSQL_DB="+c.mysqlDB,
		"R_MYSQL_TABLE=orders",
	)

	// run
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Resp{
				Success:   false,
				MsgUser:   "The report generation took too long and was aborted.",
				MsgSystem: "R script timed out.",
			}
		}
		return Resp{
			Success:   false,
			MsgUser:   "An error occurred while generating the report.",
			MsgSystem: fmt.Sprintf("R script error: %s", err.Error()),
		}
	}

	var resp Resp
	if err = json.Unmarshal(out, &resp); err != nil {
		return Resp{
			Success:   false,
			MsgUser:   "An error occurred while generating the report.",
			MsgSystem: fmt.Sprintf("JSON unmarshal error: %v, output: %s", err, string(out)),
		}
	}

	return resp
}

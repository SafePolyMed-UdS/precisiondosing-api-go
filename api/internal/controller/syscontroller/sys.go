package syscontroller

import (
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/utils/srvstats"

	"github.com/gin-gonic/gin"
)

type SysController struct {
	Meta cfg.MetaConfig
}

func New(resourceHandle *handle.ResourceHandle) *SysController {
	return &SysController{
		Meta: resourceHandle.MetaCfg,
	}
}

// @Summary		Ping the API
// @Description	Ping the API to check if it is alive.
// @Tags			System
// @Produce		json
// @Success		200	{object}	handle.jsendSuccess[PingResp]	"Response with pong message"
// @Router			/sys/ping [get]
func (sc *SysController) GetPing(c *gin.Context) {
	type PingResponse struct {
		Message string `json:"message" example:"pong"` // Message
	} // @name PingResp

	handle.Success(c, PingResponse{Message: "pong"})
}

// @Summary		Get API Info
// @Description	Get information about the API including version and query limits.
// @Tags			System
// @Produce		json
// @Success		200	{object}	handle.jsendSuccess[InfoResp]	"Response with API info"
// @router			/sys/info [get]
func (sc *SysController) GetInfo(c *gin.Context) {
	type InfoResponse struct {
		API cfg.MetaConfig `json:"meta_info"` // Meta
	} // @name InfoResp

	res := InfoResponse{
		API: sc.Meta,
	}

	handle.Success(c, res)
}

func (sc *SysController) GetServerStats(c *gin.Context) {

	cpuStats, err := srvstats.CPU()
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	memStats, err := srvstats.Memory()
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	res := struct {
		CPU srvstats.CPUStats    `json:"cpu"` // CPU stats
		Mem srvstats.MemoryStats `json:"mem"` // Memory stats
	}{
		CPU: cpuStats,
		Mem: memStats,
	}

	handle.Success(c, res)
}

func (sc *SysController) GetProcessStats(c *gin.Context) {
	processStasts, err := srvstats.Process()
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, processStasts)
}

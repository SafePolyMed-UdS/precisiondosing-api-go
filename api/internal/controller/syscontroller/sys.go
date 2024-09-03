package syscontroller

import (
	"net/http"
	"precisiondosing-api-go/cfg"
	"precisiondosing-api-go/internal/handle"

	"github.com/gin-gonic/gin"
)

type SysController struct {
	Meta cfg.MetaConfig
}

func NewSysController(resourceHandle *handle.ResourceHandle) *SysController {
	return &SysController{
		Meta: resourceHandle.MetaCfg,
	}
}

func (sc *SysController) GetPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (sc *SysController) GetInfo(c *gin.Context) {

	res := struct {
		API cfg.MetaConfig `json:"meta_info"`
	}{
		sc.Meta,
	}

	c.JSON(http.StatusOK, res)
}

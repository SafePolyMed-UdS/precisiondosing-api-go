package modelcontroller

import (
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/pbpk"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	Models []pbpk.ModelDefinition
}

func New(models []pbpk.ModelDefinition) *ModelController {
	return &ModelController{
		Models: models,
	}
}

// @Summary		List available models
// @Description	__Authentication required__
// @Description	Retrieve a list of all available PBPK models.
// @Tags			Models
// @Produce		json
// @Success		200	{object}	handle.jsendSuccess[ModelsResp]	"List of models"
// @Security		Bearer
// @Router			/models [get]
func (mc *ModelController) GetModels(c *gin.Context) {
	type ModelResponse struct {
		Models []pbpk.ModelDefinition `json:"models"` // List of models
	} // @name ModelsResp

	res := ModelResponse{
		Models: mc.Models,
	}

	handle.Success(c, res)
}

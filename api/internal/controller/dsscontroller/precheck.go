package dsscontroller

import (
	"precisiondosing-api-go/internal/handle"

	"github.com/gin-gonic/gin"
)

func (sc *DSSController) PostPrecheck(c *gin.Context) {
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	result, precheckErr := sc.Prechecker.Check(patientData)
	if precheckErr != nil {
		handle.ServerError(c, precheckErr)
		return
	}

	handle.Success(c, result)
}

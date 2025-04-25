package dsscontroller

import (
	"precisiondosing-api-go/internal/handle"
	"precisiondosing-api-go/internal/utils/precheck"

	"github.com/gin-gonic/gin"
)

func (sc *DSSController) PostPrecheck(c *gin.Context) {
	patientData, err := sc.readPatientData(c)
	if err != nil {
		handle.BadRequestError(c, err.Error())
		return
	}

	preCheck := precheck.New(sc.IndibidualsDB, sc.ABDATA)
	result, err := preCheck.Check(patientData)
	if err != nil {
		handle.ServerError(c, err)
		return
	}

	handle.Success(c, result)
}

package v1

import (
	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AuthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, utils.Response{Data: "Authentication check successful"})
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, utils.Response{Data: "OK"})
}

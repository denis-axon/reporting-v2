package v1

import (
	"net/http"

	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/gin-gonic/gin"
)

func GetOrgClusters(c *gin.Context) {
	orgId := c.Query("orgId")
	if orgId == "" {
		c.JSON(http.StatusBadRequest, utils.Response{Error: "org not specified or org FID not found"})
		return
	}

	// cl, err := axonserver.GetClusters(orgId)
	err := metrics.GetClusters(orgId)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})

}

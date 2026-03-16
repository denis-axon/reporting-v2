package v1

import (
	"net/http"

	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/components/metrics"
	"github.com/gin-gonic/gin"
)

func GetOrgClusters(c *gin.Context) {
	orgId := c.Query("orgId")
	clusterType := c.Query("clusterType")
	clusterName := c.Query("clusterName")
	if orgId == "" {
		c.JSON(http.StatusBadRequest, utils.Response{Error: "org not specified or org FID not found"})
		return
	}

	err, snapshot := metrics.GetCassandraSnapshot(orgId, clusterType, clusterName)

	// cl, err := axonserver.GetClusters(orgId)
	// allDetails, err := metrics.GetClusters(orgId)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting clusters for org %s: %s", orgId, err)
		return
	}

	// c.JSON(http.StatusOK, gin.H{"clusters": allDetails})
	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

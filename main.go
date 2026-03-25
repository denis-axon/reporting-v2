package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	apiv1 "github.com/denis-axon/reporting-v2/api/v1"
	"github.com/denis-axon/reporting-v2/api/v1/utils"
	"github.com/denis-axon/reporting-v2/config"
	"github.com/gin-gonic/gin"

	log "bitbucket.org/digitalisio/go/logger"
)

func main() {
	// Enable pprof on localhost port 6060
	go func() {
		_ = http.ListenAndServe("127.0.0.1:6060", nil)
	}()
	r := gin.New()
	setupRoutes(r)

	// Start the HTTP server in the background
	go func() {
		err := r.Run(config.GetInstance().ListenAddress) // listen and serve on 0.0.0.0:8081 by default
		if err != nil {
			log.Error(fmt.Sprintf("HTTP server exited with error: %s", err.Error()))
		}
	}()

	// Wait for an exit signal
	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)
	signal.Notify(signalChan, syscall.SIGTERM, os.Interrupt)
	s := <-signalChan
	log.Warn(fmt.Sprintf("captured %v signal, stopping gracefully...", s))
	signal.Stop(signalChan)
}

func authenticationCheck(c *gin.Context) {
	fmt.Println("Authentication check!")
	suppliedToken := c.GetHeader("X-AxonOps-Auth")
	expectedToken := config.AuthToken()
	// fmt.Printf("Authentication check: supplied token '%s', expected token '%s'\n", suppliedToken, expectedToken)
	if suppliedToken != expectedToken {
		c.JSON(http.StatusUnauthorized, utils.Response{Error: "Unauthorized"})
		c.Abort()
	}
}

func setupRoutes(r *gin.Engine) {
	// Health check endpoints
	r.GET("/healthz", apiv1.HealthCheck)

	// v1 API
	v1 := r.Group("/v1")
	// we don't use authentication for now because we'll need this to work
	// for both SaaS and on-prem installs and the authentication works differently in these environments
	// v1.Use(authenticationCheck)

	// Authenticated dummy endpoint
	v1.GET("/authcheck", apiv1.AuthCheck)

	v1.GET("/reporting", apiv1.GeneratePDF)
	v1.GET("/clusters", apiv1.GetOrgClusters)
	v1.POST("/events", apiv1.GetOrgClusterEvents)

	// ensure there is a newline in the output because it doesn't always display correctly and it's annoying!
	fmt.Println()
}

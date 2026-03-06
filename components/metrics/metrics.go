package metrics

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"bitbucket.org/digitalisio/axon-metrics-go-client/client"
	"github.com/denis-axon/reporting-v2/config"
	"go.uber.org/zap"
)

var (
	clientMu       sync.Mutex
	clientInstance *client.Client
)

func GetClient() *client.Client {
	clientMu.Lock()
	defer clientMu.Unlock()
	return clientInstance
}

func Init(org string) error {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	w := bytes.NewBuffer(nil)
	cfg := config.GetInstance()

	// TODO: add an ability to switch between Regular and SAML modes URLs based on org settings in Cloud API
	err := cfg.AxonServerUrlTemplate.Execute(w, map[string]string{"Org": org})
	if err != nil {
		return err
	}

	baseURl := w.String()

	c, err := client.New(client.Options{
		BaseURL:   baseURl,
		AuthToken: "Bearer " + cfg.AuthToken,
		Timeout:   10 * time.Second,
		Logger:    logger,
	})
	if err != nil {
		return err
	}
	logger.Info("Metrics client initialized successfully", zap.Any("client", c))

	clientMu.Lock()
	clientInstance = c
	clientMu.Unlock()

	return nil
}

func Healthy() (error, bool) {
	c := GetClient()
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/v1/healthz").
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err, false
	}

	return nil, resp.StatusCode == 200
}

func GetChartImage() ([]byte, error) {
	c := GetClient()
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/dash/chartImage").
		WithQueryParam("org", "testorg3").
		WithQueryParam("clusterType", "cassandra").
		WithQueryParam("cluster", "test41cluster").
		WithQueryParam("width", "800").
		WithQueryParam("height", "400").
		WithQueryParam("timeZone", "Asia/Makassar").
		WithQueryParam("from", "1772678548").
		WithQueryParam("to", "1772721748").
		WithQueryParam("widgetUuid", "c11b97f0-6b2e-40cd-abc6-b721e38778b9").
		Build()

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Data, nil
}

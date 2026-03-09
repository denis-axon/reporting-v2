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
	clientMu sync.Mutex
	clients  = make(map[string]*client.Client)
)

func GetClient(org string) *client.Client {
	clientMu.Lock()
	defer clientMu.Unlock()
	return clients[org]
}

func Init(org string) error {
	clientMu.Lock()
	if _, exists := clients[org]; exists {
		clientMu.Unlock()
		return nil
	}
	clientMu.Unlock()

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
	logger.Info("Metrics client initialized successfully", zap.String("org", org), zap.Any("client", c))

	clientMu.Lock()
	clients[org] = c
	clientMu.Unlock()

	return nil
}

func Healthy(org string) (error, bool) {
	c := GetClient(org)
	if c == nil {
		return fmt.Errorf("metrics client not initialized for org %s", org), false
	}
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

func GetChartImage(org string, clusterName string, clusterType string, from string, to string, timeZone string, widgetUuid string) ([]byte, error) {
	c := GetClient(org)
	if c == nil {
		return nil, fmt.Errorf("metrics client not initialized for org %s", org)
	}
	ctx := context.Background()
	req := client.NewRequest().
		WithMethod("GET").
		WithPath("/dashboard/api/dash/chartImage").
		WithQueryParam("org", org).
		WithQueryParam("cluster", clusterName).
		WithQueryParam("clusterType", clusterType).
		WithQueryParam("width", "800").
		WithQueryParam("height", "400").
		WithQueryParam("timeZone", timeZone).
		WithQueryParam("from", from).
		WithQueryParam("to", to).
		WithQueryParam("widgetUuid", widgetUuid).
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

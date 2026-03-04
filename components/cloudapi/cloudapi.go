package cloudapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/denis-axon/reporting-v2/components/httputil"
	"github.com/denis-axon/reporting-v2/components/logger"
	"github.com/denis-axon/reporting-v2/config"
)

var log = logger.Get()

var cloudClientInstance *CloudClient
var cloudClientInstanceOnce sync.Once

type Response struct {
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

type RequestError struct {
	StatusCode int
	Message    string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request failed with status code %d: %s", e.StatusCode, e.Message)
}

type CloudClient struct {
	Endpoint  string
	Proxy     string
	AuthToken string

	CallAttempts int
	client       *http.Client
}

func GetClient() (*CloudClient, error) {
	var retErr error
	cloudClientInstanceOnce.Do(func() {
		cfg := config.GetInstance()
		log.Infof("CloudAPIEndpoint: %s", cfg.CloudAPIEndpoint)
		log.Infof("CloudAPIProxy: %s", cfg.CloudAPIProxy)
		log.Infof("CloudAPIToken: %s", cfg.CloudAPIToken)
		c := &CloudClient{
			Endpoint:     strings.TrimRight(cfg.CloudAPIEndpoint, "/"),
			Proxy:        cfg.CloudAPIProxy,
			AuthToken:    cfg.CloudAPIToken,
			CallAttempts: 4,
		}

		rt := &http.Transport{}
		if strings.Trim(c.Proxy, " ") != "" {
			proxyURL, err := url.Parse(c.Proxy)
			if err != nil {
				retErr = err
				return
			}
			rt.Proxy = http.ProxyURL(proxyURL)
		}
		rt.TLSHandshakeTimeout = 2 * time.Second
		rt.ResponseHeaderTimeout = 2 * time.Second
		c.client = &http.Client{Transport: rt, Timeout: 10 * time.Second}
		cloudClientInstance = c
	})
	return cloudClientInstance, retErr
}

// SimpleRequest performs a GET to the given relative URL and puts the Data response into dest
func SimpleRequest(relUrl string, dest interface{}) error {
	return DoRequest(relUrl, "GET", dest, nil, nil)
}

// DoRequest performs a request to Cloud API (with retries) and puts the Data response into dest
func DoRequest(relUrl string, method string, dest interface{}, payload interface{}, headers map[string]string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["X-AxonOps-Auth"] = config.GetInstance().CloudAPIToken
	res, err := httputil.DoRequestWithRetries(httputil.JoinUrl(c.Endpoint, relUrl), method, payload, headers, c.CallAttempts)
	if err != nil {
		return err
	}

	if dest != nil {
		var response *Response
		err = json.Unmarshal(res, &response)
		if err != nil {
			return err
		}

		err = reUnmarshalJson(response.Data, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func reUnmarshalJson(src interface{}, dest interface{}) error {
	js, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, dest)
}

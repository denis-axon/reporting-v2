package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "bitbucket.org/digitalisio/go/logger"
)

type RequestError struct {
	StatusCode int
	Message    string
}

func (e RequestError) Error() string {
	return fmt.Sprintf("request failed with status code %d: %s", e.StatusCode, e.Message)
}

var client *http.Client
var clientMu sync.Mutex

func getClient() *http.Client {
	clientMu.Lock()
	defer clientMu.Unlock()
	if client == nil {
		client = &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   15 * time.Second,
		}
	}
	return client
}

func DoRequestWithRetries(fullUrl string, method string, payload interface{}, headers map[string]string, maxAttempts int) ([]byte, error) {
	var body []byte
	var err error
	for i := 1; i <= maxAttempts; i++ {
		body, err = DoRequest(fullUrl, method, payload, headers)
		if err == nil {
			return body, nil
		} else {
			var rErr *RequestError
			var uErr *url.Error
			if errors.As(err, &rErr) && rErr.StatusCode >= 400 && rErr.StatusCode < 500 {
				// 4xx Bad Request or Not Found error. Don't retry these
				log.Error(fmt.Sprintf("HTTP request to %s failed with response code %d", fullUrl, rErr.StatusCode))
				return nil, rErr
			} else if errors.As(err, &uErr) {
				log.Error(fmt.Sprintf("HTTP request to %s failed with URL error: %d", fullUrl, uErr.Err))
				return nil, err
			} else {
				log.Error("HTTP request failed: " + err.Error())
			}
		}
		log.Warn(fmt.Sprintf("HTTP client retry %d", i))
		time.Sleep(time.Duration(i) * 500 * time.Millisecond)
	}

	if err == nil {
		err = errors.New("ran out of retries performing HTTP request")
	}
	return nil, err
}

// DoRequest Performs an HTTP request and returns the response body
func DoRequest(fullUrl string, method string, payload interface{}, headers map[string]string) ([]byte, error) {
	var payloadBytes []byte
	var err error
	// if len(payload) != 0 {
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	} else {
		payloadBytes = nil
	}

	req, err := http.NewRequest(method, fullUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := getClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %s", err.Error())
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, &RequestError{
			StatusCode: res.StatusCode,
			Message:    string(body),
		}
	}
	return body, nil
}

func JoinUrl(parts ...string) string {
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			cleanParts = append(cleanParts, p)
		}
	}
	return strings.Join(cleanParts, "/")
}

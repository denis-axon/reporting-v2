package axonserver

import (
	"bytes"
	"errors"

	"github.com/denis-axon/reporting-v2/components/httputil"
	"github.com/denis-axon/reporting-v2/config"
)
	
func doApiCall(org, apiCall string) ([]byte, error) {
	// testorg1 API key for any cluster
	tok := config.GetInstance().AuthToken
	if tok == "" {
		return nil, errors.New("no auth token provided")
	}

	w := bytes.NewBuffer(nil)
	err := config.GetInstance().AxonServerUrlTemplate.Execute(w, map[string]string{"Org": org})
	if err != nil {
		return nil, err
	}
	return httputil.DoRequestWithRetries(httputil.JoinUrl(w.String(), apiCall),
		"GET", nil, map[string]string{"Authorization": "Bearer " + tok}, 3)
}

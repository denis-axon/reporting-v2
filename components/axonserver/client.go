package axonserver

import (
	"bytes"
	"errors"

	"github.com/denis-axon/reporting-v2/components/httputil"
	"github.com/denis-axon/reporting-v2/config"
)

func doApiCall(org, apiCall string) ([]byte, error) {
	// here should be a AxonSaas API key for the particular org and cluster
	// TODO: check the token if it's for the testorg3
	tok := config.GetInstance().AuthToken
	if tok == "" {
		return nil, errors.New("no auth token provided")
	}

	w := bytes.NewBuffer(nil)
	// TODO: add an ability to switch between Regular and SAML modes URLs based on org settings in Cloud API
	err := config.GetInstance().AxonServerUrlTemplateSaml.Execute(w, map[string]string{"Org": org})
	if err != nil {
		return nil, err
	}
	return httputil.DoRequestWithRetries(httputil.JoinUrl(w.String(), apiCall),
		"GET", nil, map[string]string{"Authorization": "Bearer " + tok}, 3)
}

package cloudapi

import (
	"strings"
)

type OrganizationDetails struct {
	OrgName                   string   `json:"orgName"` // Org name (ID)
	OrgId                     string   `json:"orgID"`   // Org FID
	NodesLimit                int      `json:"nodesLimit"`
	AgentKeys                 []string `json:"agentKeys"`
	MaxAccounts               int      `json:"maxAccounts"`
	SubscriptionType          string   `json:"subscriptionType"`
	ManualPayments            bool     `json:"manualPayments"`
	SupportedProducts         []string `json:"supportedProducts"`
	CassandraSupport          bool     `json:"cassandraSupport"`
	StartDate                 int64    `json:"startDate"`       // UNIX timestamp
	ExpiresAt                 int64    `json:"expiresAt"`       // UNIX timestamp
	LastInstallTime           int64    `json:"lastInstallTime"` // UNIX timestamp
	DashRequestedVersion      string   `json:"dashRequestedVersion"`
	DashInstalledVersion      string   `json:"dashInstalledVersion"`
	ServerRequestedVersion    string   `json:"serverRequestedVersion"`
	ServerInstalledVersion    string   `json:"serverInstalledVersion"`
	AgentVersion              string   `json:"agentVersion"`
	CassandraTicketPriorities string   `json:"cassandraTicketPriorities"` // List of P1,P2 etc.
	SamlAllowed               bool     `json:"samlAllowed"`
	MfaRequired               bool     `json:"mfaRequired"`
	SupportModel              string   `json:"supportModel"`
	LicenseType               string   `json:"licenseType"`
	ProvisioningEnabled       bool     `json:"provisioningEnabled"`
	FirstAgentConnected       int64    `json:"firstAgentConnected"` // UNIX timestamp
}

type OrgCluster struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status int    `json:"status"`
	Nodes  int    `json:"nodes"`
}

func ListOrgs() ([]OrganizationDetails, error) {
	var orgs []OrganizationDetails
	err := SimpleRequest("/v1/orgs", &orgs)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

func GetOrg(orgId string) (OrganizationDetails, error) {
	var o OrganizationDetails
	err := SimpleRequest("/v1/orgs/"+orgId, &o)
	if err != nil {
		return o, err
	}
	return o, nil
}

func DeleteOrg(orgName string) error {
	err := DoRequest("/v1/orgs/"+orgName, "DELETE", nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func GetOrgClusters(orgName string) ([]OrgCluster, error) {
	var clusters []OrgCluster
	err := SimpleRequest("/v1/orgs/"+orgName+"/clusters", &clusters)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func GetUserProfile(userId string) (map[string]interface{}, error) {
	var profile map[string]interface{}
	err := SimpleRequest("/v1/userProfile/"+userId, &profile)
	if err != nil {
		log.Infof("GetUserProfile err %s", err)
		// users who have never sign in to their orgs don't have profile entries
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return profile, err
	}
	return profile, nil
}

func UpdateUserProfile(email string, profile *map[string]interface{}) error {
	var httpResponse map[string]interface{}
	err := DoRequest("/v1/userProfile/"+email, "PUT", &httpResponse, profile, nil)
	if err != nil {
		return err
	}
	return nil
}

func DeleteOrgFromMetadataAndUpdateUser(email string, orgNameToRemove string) error {
	// get user profile
	profile, err := GetUserProfile(email)
	if err != nil {
		return err
	}

	// remove orgMeta from profile
	if profile != nil {
		if orgList, ok := (profile)["orgs"]; ok {
			if orgs, ok := orgList.([]interface{}); ok && len(orgs) > 0 {

				// Create a new slice to store the filtered organizations
				newOrgs := make([]interface{}, 0, len(orgs))

				log.Infof("orgs CloudAPI before len: %d", len(orgs))

				// Filter out the organization to remove
				for _, o := range orgs {
					// remove orgMeta from CloudAPI profile
					orgData, ok := o.(map[string]interface{})
					if !ok {
						continue
					}
					orgName, ok := orgData["orgName"].(string)
					if !ok {
						continue
					}
					if orgName != orgNameToRemove {
						newOrgs = append(newOrgs, o)
					}
				}
				log.Infof("orgs CloudAPI after len: %d", len(newOrgs))

				// update user metadata against CloudAPI
				updatedProfil := map[string]interface{}{
					"orgs": newOrgs,
				}
				log.Infof("Updated profile: %+v", updatedProfil)

				err = UpdateUserProfile(email, &updatedProfil)
				if err != nil {
					return err
				}
				log.Infof("Updated successfully: %s", email)
			}
		}
	}

	return nil
}

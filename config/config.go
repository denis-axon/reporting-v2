// ******************************************************
// Copyright (C) digitalis.io Limited <info@digitalis.io> - All Rights Reserved
//
// This file is the property of digitalis.io Limited
// Unauthorized copying of this file, via any medium is strictly prohibited
// ******************************************************

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	log "bitbucket.org/digitalisio/go/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ListenAddress string

	// Axon Server authentication token (shared secret with axon-server, used for both regular and SAML modes)
	AuthToken string

	CloudAPIEndpoint string
	CloudAPIProxy    string
	// Cloud API authentication token (shared secret with axon-server)
	CloudAPIToken string

	// Vault connection details
	VaultHost     string
	VaultRoleID   string
	VaultSecretID string

	// Cassandra connection details
	CassandraHosts               []string
	CassandraDc                  string
	CassandraKeyspace            string
	CassandraUsername            string
	CassandraPassword            string
	CassandraTls                 bool
	CassandraTlsCa               string
	CassandraTlsClientCert       string
	CassandraTlsClientKey        string
	CassandraTlsHostVerification bool
	CassandraTimeout             time.Duration

	// Base URL for agent downloads
	AgentDownloadUrlBase string

	// Fortifi API credentials
	FortifiOrgFid string
	FortifiUser   string
	FortifiKey    string

	// Airflow engine details
	AirflowURL      string
	AirflowUser     string
	AirflowPassword string

	// Prometheus helper config
	PrometheusUrl string

	RefreshCloudInstancesCache bool

	// Audit config
	AuditLogBQEnableBool  bool
	AuditLogBQCredentials string
	AuditLogBQDataSet     string
	AuditLogBQProject     string
	AuditLogBQTable       string

	AxonServerUrlTemplate     *template.Template
	DisableClusterListRefresh bool
}

var confMu sync.Mutex
var instance *Config

func GetInstance() *Config {
	confMu.Lock()
	defer confMu.Unlock()
	if instance == nil {
		instance = loadConfigFromEnv()
	}
	return instance
}

func loadConfigFromEnv() *Config {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = "0.0.0.0:8081"
	}

	var cassHosts []string
	hostsRaw := strings.Split(os.Getenv("CASSANDRA_HOSTS"), ",")
	for _, h := range hostsRaw {
		tidied := strings.TrimSpace(h)
		if tidied != "" {
			cassHosts = append(cassHosts, tidied)
		}
	}
	// Cassandra timeout in milliseconds
	cassTimeout := 10000
	cassTimeOutStr := os.Getenv("CASSANDRA_TIMEOUT")
	if cassTimeOutStr != "" {
		v, err := strconv.ParseInt(cassTimeOutStr, 10, 64)
		if err != nil {
			log.Warn(fmt.Sprintf("Invalid value set for CASSANDRA_TIMEOUT: %s", cassTimeOutStr))
		}
		cassTimeout = int(v)
	}

	downloadBaseUrl := os.Getenv("AGENT_DOWNLOAD_BASE_URL")
	if downloadBaseUrl == "" {
		downloadBaseUrl = "https://agents.axonops.cloud/downloads"
	}

	prometheusUrl := os.Getenv("PROMETHEUS_URL")
	if prometheusUrl == "" {
		prometheusUrl = "http://prom-stack-kube-prometheus-prometheus.monitoring:9090"
	}

	auditLogBQEnable := os.Getenv("AUDIT_LOG_ENABLE")
	auditLogBQEnableBool, err := strconv.ParseBool(auditLogBQEnable)
	if err != nil {
		log.Error(fmt.Sprintf("Unable to parse AUDIT_LOG_ENABLE as bool: %s", auditLogBQEnable))
		log.Warn("Setting AUDIT_LOG_ENABLE to false")
		auditLogBQEnableBool = true
	}
	if auditLogBQEnable == "" || !auditLogBQEnableBool {
		log.Warn("Setting AUDIT_LOG_ENABLE to false")
		auditLogBQEnableBool = false
	}

	auditLogBQCredentials := os.Getenv("AUDIT_LOG_BQ_CREDENTIALS")
	if auditLogBQCredentials == "" && auditLogBQEnableBool {
		log.Fatal("Audit log enabled but AUDIT_LOG_BQ_CREDENTIALS not set")
	}

	auditLogBQDataSet := os.Getenv("AUDIT_LOG_BQ_DATASET")
	if auditLogBQDataSet == "" && auditLogBQEnableBool {
		log.Fatal("Audit log enabled but AUDIT_LOG_BQ_DATASET not set")
	}

	auditLogBQProject := os.Getenv("AUDIT_LOG_BQ_PROJECT")
	if auditLogBQProject == "" && auditLogBQEnableBool {
		log.Fatal("Audit log enabled but AUDIT_LOG_BQ_PROJECT not set")
	}

	auditLogBQTable := os.Getenv("AUDIT_LOG_BQ_TABLE")
	if auditLogBQTable == "" && auditLogBQEnableBool {
		log.Fatal("Audit log enabled but AUDIT_LOG_BQ_TABLE not set")
	}

	// Default this to true and then override in local dev environments so we are not constantly
	// performing large cache refreshes.
	refreshCloudInstancesCache := os.Getenv("REFRESH_CLOUD_INSTANCES_CACHE")
	refreshCloudInstancesCacheBool, err := strconv.ParseBool(refreshCloudInstancesCache)
	if err != nil {
		log.Warn(fmt.Sprintf("Unable to parse REFRESH_CLOUD_INSTANCES_CACHE as bool: %s", refreshCloudInstancesCache))
		log.Warn("Setting REFRESH_CLOUD_INSTANCES_CACHE to true")
		refreshCloudInstancesCacheBool = true
	}

	if refreshCloudInstancesCache == "" {
		refreshCloudInstancesCacheBool = true
	}

	axonServerUrlStr := os.Getenv("AXON_SERVER_URL_SAML_MODE_TEMPLATE")
	if axonServerUrlStr == "" {
		// the URL below does not work outside of the k8s
		// axonServerUrlStr = "http://axonops-axon-server.cst-{{.Org}}:8080/api/v1"

		// TODO: add a way to switch between regular and SAML mode URLs
		// axonServerUrlStr = "https://dash.axonopsdev.com/{{.Org}}/api/v1"

		axonServerUrlStr = "https://{{.Org}}.axonopsdev.com/dashboard/api/v1"
	}
	axonServerUrlTemplate, err := template.New("url").Parse(axonServerUrlStr)
	if err != nil {
		log.Fatal("Error parsing axon-server URL template '" + axonServerUrlStr + "': " + err.Error())
	}

	disableClusterListRefresh := os.Getenv("DISABLE_CLUSTER_LIST_REFRESH")
	disableClusterListRefreshBool, err := strconv.ParseBool(disableClusterListRefresh)
	if err != nil {
		disableClusterListRefreshBool = false
	}

	return &Config{
		AuthToken: os.Getenv("AUTH_TOKEN_TESTORG3"),

		CloudAPIToken:    os.Getenv("CLOUD_API_TOKEN"),
		CloudAPIEndpoint: os.Getenv("CLOUD_API_ENDPOINT"),
		CloudAPIProxy:    os.Getenv("CLOUD_API_PROXY"),

		ListenAddress: listenAddress,

		VaultHost:     os.Getenv("VAULT_HOST"),
		VaultRoleID:   os.Getenv("VAULT_ROLE_ID"),
		VaultSecretID: os.Getenv("VAULT_SECRET_ID"),

		CassandraHosts:               cassHosts,
		CassandraDc:                  os.Getenv("CASSANDRA_DC"),
		CassandraKeyspace:            os.Getenv("CASSANDRA_KEYSPACE"),
		CassandraUsername:            os.Getenv("CASSANDRA_USERNAME"),
		CassandraPassword:            os.Getenv("CASSANDRA_PASSWORD"),
		CassandraTls:                 os.Getenv("CASSANDRA_TLS") == "1",
		CassandraTlsCa:               os.Getenv("CASSANDRA_TLS_CA"),
		CassandraTlsClientCert:       os.Getenv("CASSANDRA_TLS_CLIENT_CERT"),
		CassandraTlsClientKey:        os.Getenv("CASSANDRA_TLS_CLIENT_KEY"),
		CassandraTlsHostVerification: os.Getenv("CASSANDRA_TLS_HOST_VERIFICATION") != "0", // defaults to true unless explicitly disabled
		CassandraTimeout:             time.Duration(cassTimeout) * time.Millisecond,

		AgentDownloadUrlBase: downloadBaseUrl,

		FortifiOrgFid: os.Getenv("FORTIFI_ORG_FID"),
		FortifiUser:   os.Getenv("FORTIFI_USER"),
		FortifiKey:    os.Getenv("FORTIFI_KEY"),

		AirflowURL:      os.Getenv("AIRFLOW_URL"),
		AirflowUser:     os.Getenv("AIRFLOW_USER"),
		AirflowPassword: os.Getenv("AIRFLOW_PASSWORD"),

		PrometheusUrl: prometheusUrl,

		RefreshCloudInstancesCache: refreshCloudInstancesCacheBool,

		AuditLogBQEnableBool:  auditLogBQEnableBool,
		AuditLogBQCredentials: auditLogBQCredentials,
		AuditLogBQProject:     auditLogBQProject,
		AuditLogBQTable:       auditLogBQTable,
		AuditLogBQDataSet:     auditLogBQDataSet,

		AxonServerUrlTemplate:     axonServerUrlTemplate,
		DisableClusterListRefresh: disableClusterListRefreshBool,
	}
}

func AuthToken() string {
	cfg := GetInstance()
	if cfg.AuthToken == "" {
		log.Fatal("No auth token has been set. Restart with the AUTH_TOKEN env var configured.")
	}
	return cfg.AuthToken
}

// PrintConfig ...
func printConfig(cfg interface{}) error {
	log.Debug("AxonOps Cloud API starting with the following parameters:")
	d, err := yaml.Marshal(&cfg)
	if err != nil {
		log.Fatal("error: %v", zap.Error(err))
	}
	log.Debug(string(d))
	return nil
}

package conf

import (
	"os"

	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const default_vault_secret_path string = "/var/run/secrets/vaultproject.io/secret.json"
const default_consul_token_path string = "consul/creds/readonly"
const default_consul_config_root string = "config/kube-pod-decorator"
const default_loglevel string = "info"
const default_ttl int = 600
const default_lease_renewal_percentage int = 75
const default_lease_failure_retry_interval int = 10

// Specification is the basic configuration injected to the wrapper process
type Specification struct {
	ApplicationName                       string
	ConsulConfigRoot                      string
	LogLevel                              string
	VaultSecretPath                       string
	ConsulTokenPath                       string
	VaultLeaseDurationSeconds             int
	VaultRenewFailureRetryIntervalSeconds int
	VaultLeaseRenewalPercentage           int
}

//InjectionDefinition represente the structure of the consol config file
type InjectionDefinition struct {
	Key       string            `yaml:"key"`
	Env       []*EnvVar         `yaml:"env"`
	Files     []*FileSource     `yaml:"files"`
	Templates []*TemplateSource `yaml:"templates"`
	Vault     []*GenericRef     `yaml:"vault"`
	Consul    []*GenericRef     `yaml:"consul"`
}
type EnvVar struct {
	Name      string        `yaml:"name"`
	Value     string        `yaml:"value,omitempty"`
	ValueFrom *EnvVarSource `yaml:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	SecretKeyRef *SecretKeyRefSelector `yaml:"secretKeyRef,omitempty"`
	Consul       *ConsulSelector       `yaml:"consulKeyRef,omitempty"`
}
type SecretKeyRefSelector struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}
type ConsulSelector struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key,omitempty"`
}

type FileSource struct {
	Name        string `yaml:"name"`
	Destination string `yaml:"destination"`
}
type TemplateSource struct {
	Name        string    `yaml:"name"`
	Destination string    `yaml:"destination"`
	Env         []*EnvVar `yaml:"env"`
}
type GenericRef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

func (s *Specification) validate() {
	if len(s.ApplicationName) == 0 {
		logger.GetLogger().Fatal("ApplicationName can not be empty")
	}

	if len(s.LogLevel) == 0 {
		s.LogLevel = "info"
	}

	logger.GetLogger().Debugf("Config: %v", s)
}

func GetSpecification() Specification {
	s := Specification{
		VaultSecretPath:                       default_vault_secret_path,
		ConsulTokenPath:                       default_consul_token_path,
		ConsulConfigRoot:                      default_consul_config_root,
		VaultLeaseDurationSeconds:             default_ttl,
		VaultLeaseRenewalPercentage:           default_lease_renewal_percentage,
		VaultRenewFailureRetryIntervalSeconds: default_lease_failure_retry_interval,
		LogLevel: default_loglevel,
	}
	err := envconfig.Process("k8sPodDecorator", &s)
	if err != nil {
		logger.GetLogger().Fatal(err)
	}

	s.validate()

	return s
}

func GetInjectionDefinition(jsonBlob []byte) InjectionDefinition {
	var definition InjectionDefinition
	err := yaml.Unmarshal([]byte(os.ExpandEnv(string(jsonBlob))), &definition)
	if err != nil {
		logger.GetLogger().Fatal(err)
	}
	return definition
}

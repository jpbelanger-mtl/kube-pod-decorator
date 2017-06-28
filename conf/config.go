package conf

import (
	"gopkg.in/yaml.v2"

	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
	"github.com/kelseyhightower/envconfig"
)

const default_vault_secret_path string = "/var/run/secrets/vaultproject.io/secret.json"
const default_consul_token_path string = "consul/creds/readonly"

// Specification is the basic configuration injected to the wrapper process
type Specification struct {
	ApplicationName                       string
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

	if s.VaultLeaseDurationSeconds == 0 {
		s.VaultLeaseDurationSeconds = 600
	}

	if s.VaultRenewFailureRetryIntervalSeconds == 0 {
		s.VaultRenewFailureRetryIntervalSeconds = 10
	}

	if s.VaultLeaseRenewalPercentage == 0 {
		s.VaultLeaseRenewalPercentage = 75
	}

	logger.GetLogger().Infof("Config: %v", s)
}

func GetSpecification() Specification {
	s := Specification{
		VaultSecretPath: default_vault_secret_path,
		ConsulTokenPath: default_consul_token_path,
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
	err := yaml.Unmarshal(jsonBlob, &definition)
	if err != nil {
		logger.GetLogger().Fatal(err)
	}
	return definition
}

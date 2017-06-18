package conf

import (
	"log"

	"gopkg.in/yaml.v2"

	"github.com/kelseyhightower/envconfig"
)

// Specification is the basic configuration injected to the wrapper process
type Specification struct {
	VaultHost       string
	ApplicationName string
	VaultSecretPath string
	ConsulTokenPath string
}

//InjectionDefinition represente the structure of the consol config file
type InjectionDefinition struct {
	Key       string           `yaml:"key"`
	Env       []EnvVar         `yaml:"env"`
	Files     []FileSource     `yaml:"files"`
	Templates []TemplateSource `yaml:"templates"`
	Vault     []GenericRef     `yaml:"vault"`
	Consul    []GenericRef     `yaml:"consul"`
}
type EnvVar struct {
	Name      string       `yaml:"name"`
	Value     string       `yaml:"value,omitempty"`
	ValueFrom EnvVarSource `yaml:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	SecretKeyRef SecretKeyRefSelector `yaml:"secretGeneric,omitempty"`
	Consul       ConsulSelector       `yaml:"consul,omitempty"`
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
	Env []EnvVar `yaml:"env"`
	FileSource
}
type GenericRef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

func GetSpecification() Specification {
	s := Specification{
		VaultSecretPath: "/var/run/secrets/vaultproject.io/secret.json",
		ConsulTokenPath: "consul/creds/readonly",
	}
	err := envconfig.Process("k8sPodDecorator", &s)
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func GetInjectionDefinition(jsonBlob []byte) InjectionDefinition {
	var definition InjectionDefinition
	err := yaml.Unmarshal(jsonBlob, &definition)
	if err != nil {
		log.Fatal(err)
	}
	return definition
}

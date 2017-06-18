package vault

import (
	"log"
	"strings"

	"io/ioutil"

	"../conf"
	vault "github.com/hashicorp/vault/api"
)

type VaultUtils struct {
	config  conf.Specification
	leaseID string
	client  *vault.Client
}

func NewVaultUtils(
	config conf.Specification,
) *VaultUtils {
	vu := VaultUtils{
		config: config,
	}
	vu.init()
	return &vu
}

func (vf *VaultUtils) init() {
	vaultConfig := vault.Config{Address: vf.config.VaultHost}
	client, err := vault.NewClient(&vaultConfig)
	if err != nil {
		log.Fatal(err)
	}
	token, err := ioutil.ReadFile(vf.config.VaultSecretPath)
	if err != nil {
		log.Fatal(err)
	}
	client.SetToken(strings.TrimSpace(string(token)))

	secret, err := client.Auth().Token().LookupSelf()
	if err != nil {
		log.Fatal(err)
	}

	vf.leaseID = secret.LeaseID
	vf.client = client
}

func (vf *VaultUtils) Renew() *error {
	_, err := vf.client.Sys().Renew(vf.leaseID, 600)
	if err != nil {
		return &err
	}
	return nil
}

func (vf *VaultUtils) Read(reference conf.GenericRef) (*vault.Secret, error) {
	log.Printf("Fetching secret at %v", reference.Path)
	secret, err := vf.client.Logical().Read(reference.Path)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

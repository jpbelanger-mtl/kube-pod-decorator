package vault

import (
	"strings"

	"io/ioutil"

	vault "github.com/hashicorp/vault/api"
	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
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
	vaultConfig := vault.DefaultConfig()
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		logger.GetLogger().Fatal(err)
	}
	token, err := ioutil.ReadFile(vf.config.VaultSecretPath)
	if err != nil {
		logger.GetLogger().Fatal(err)
	}
	client.SetToken(strings.TrimSpace(string(token)))

	secret, err := client.Auth().Token().LookupSelf()
	if err != nil {
		logger.GetLogger().Fatal(err)
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

func (vf *VaultUtils) Read(reference *conf.GenericRef) (*vault.Secret, error) {
	logger.GetLogger().Infof("Fetching secret at %v", reference.Path)
	secret, err := vf.client.Logical().Read(reference.Path)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

package vault

import (
	"fmt"
	"strings"
	"time"

	"io/ioutil"

	vault "github.com/hashicorp/vault/api"
	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
)

type VaultUtils struct {
	config        conf.Specification
	client        *vault.Client
	ShutdownCh    <-chan struct{}
	localShutdown chan bool
}

func NewVaultUtils(
	config conf.Specification,
	shutdownChan <-chan struct{},
) *VaultUtils {
	vu := VaultUtils{
		config:     config,
		ShutdownCh: shutdownChan,
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

	vf.client = client
}

func (vf *VaultUtils) Read(reference *conf.GenericRef) (*vault.Secret, error) {
	logger.GetLogger().Infof("Fetching secret at %v", reference.Path)
	secret, err := vf.client.Logical().Read(reference.Path)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (vf *VaultUtils) StartRenewal() error {
	// start renewal process when leaseDuration is at VaultLeaseRenewalPercentage
	renewalInterval := vf.config.VaultLeaseDurationSeconds / 2
	logger.GetLogger().Infof("debug %v", renewalInterval)
	c, err := time.ParseDuration(fmt.Sprintf("%vs", renewalInterval))
	if err != nil {
		return err
	}

	logger.GetLogger().Infof("Starting renewal process every %v", c)

	//Create a timer for the next execution based on the interval defined in the config
	renewTicker := time.NewTimer(c)

OUT:
	for {
		select {
		case <-vf.ShutdownCh:
			logger.GetLogger().Info("Vault renew Teardown from signal")
			break OUT

		case <-vf.localShutdown:
			logger.GetLogger().Info("Processing terminating")
			break OUT

		case <-renewTicker.C:
			//Try to renew, if it fails, use a shorter interval
			err := vf.renew()
			if err != nil {
				retryDelay, _ := time.ParseDuration(fmt.Sprintf("%vs", vf.config.VaultRenewFailureRetryIntervalSeconds))
				renewTicker = time.NewTimer(retryDelay)
				logger.GetLogger().Errorf("Error while calling renew %v, will retry in %v", err, retryDelay)
			} else {
				renewTicker = time.NewTimer(c)
				logger.GetLogger().Infof("Next lease renewal will be in %v", c)
			}
		}
	}

	return nil
}

func (vf *VaultUtils) renew() error {
	_, err := vf.client.Auth().Token().RenewSelf(vf.config.VaultLeaseDurationSeconds)
	if err != nil {
		return err
	}

	return nil
}

func (vf *VaultUtils) Revoke() error {
	logger.GetLogger().Infof("Revoking token")
	err := vf.client.Auth().Token().RevokeSelf("") //param is unused, kept for backward-comp
	if err != nil {
		return err
	}

	return nil
}

func (vf *VaultUtils) Shutdown() error {
	logger.GetLogger().Infof("Shutdown vault renewal")
	vf.localShutdown <- true
	return nil
}

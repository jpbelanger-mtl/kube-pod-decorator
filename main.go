package main

import (
	"sync"

	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/consul"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
	"github.com/jpbelanger-mtl/kube-pod-decorator/vault"
	"github.com/jpbelanger-mtl/kube-pod-decorator/wrapper"
)

func main() {
	var s = conf.GetSpecification()

	logger.InitLogger(s.LogLevel)

	shutdownCh := wrapper.MakeShutdownCh(nil)
	vaultUtils := vault.NewVaultUtils(s, shutdownCh)

	//Fetching consul token to be able to load wrapper config
	consulTokenSecret, err := vaultUtils.Read(&conf.GenericRef{Path: s.ConsulTokenPath})
	if err != nil {
		logger.GetLogger().Fatal(err)
	}

	consulToken := consulTokenSecret.Data["token"].(string)
	logger.GetLogger().Debugf("Consul token: %s", consulToken)

	consulUtils := consul.NewConsulUtils(s, consulToken)
	configKV, err := consulUtils.GetPodConfig()
	if err != nil {
		logger.GetLogger().Fatal(err)
	} else if configKV == nil {
		logger.GetLogger().Fatal("Could not fetch pod's config")
	}
	definition := conf.GetInjectionDefinition(configKV.Value)

	//Map linking secretRef to the actual Vault Secret
	secretMap := make(map[*conf.GenericRef]*vaultapi.Secret)
	consulMap := make(map[*conf.GenericRef]*consulapi.KVPair)

	//Fetching all secrets from vault
	logger.GetLogger().Info("Fetching vault secrets")
	for _, secretRef := range definition.Vault {
		secret, err := vaultUtils.Read(secretRef)
		if err != nil {
			logger.GetLogger().Errorf("error during secret fetch: %v", err)
		} else if secret == nil {
			logger.GetLogger().Errorf("No secret found at %v", secretRef.Path)
		} else {
			secretMap[secretRef] = secret
		}
	}

	//Fetching all values from consul
	logger.GetLogger().Info("Fetching consul values")
	for _, consulRef := range definition.Consul {
		kvPair, err := consulUtils.GetValue(consulRef.Path)
		if err != nil {
			logger.GetLogger().Errorf("Error while fetchin consul path %v, %v", consulRef.Path, err)
		} else if kvPair == nil {
			logger.GetLogger().Errorf("Could not find any KV at path %v", consulRef.Path)
		} else {
			consulMap[consulRef] = kvPair
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.GetLogger().Infof("Renewal gofunction terminated")
		vaultUtils.StartRenewal()
		logger.GetLogger().Infof("Terminating go routine for renewal")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.GetLogger().Infof("Wrapper gofunction terminated")
		wrapper.Wrap(&definition, secretMap, consulMap, consulUtils, shutdownCh)
		vaultUtils.Revoke()
		vaultUtils.Shutdown()
	}()

	wg.Wait()
	logger.GetLogger().Info("Exiting main wrapper process")
}

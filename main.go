package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"io/ioutil"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
	"github.com/jpbelanger-mtl/kube-pod-decorator/vault"
)

func main() {
	logger.InitLogger("info")

	var s conf.Specification = conf.GetSpecification()

	definition := test()
	vaultUtils := vault.NewVaultUtils(s)

	//Fetching consul token to be able to load wrapper config
	consulTokenSecret, err := vaultUtils.Read(conf.GenericRef{Path: s.ConsulTokenPath})
	if err != nil {
		log.Fatal(err)
	}

	consulToken := consulTokenSecret.Data["token"].(string)
	println(consulToken)

	//Map linking secretRef to the actual Vault Secret
	secretMap := make(map[conf.GenericRef]*vaultapi.Secret)

	//Fetching all secrets from vault
	for _, secretRef := range definition.Vault {
		secret, err := vaultUtils.Read(secretRef)
		if err != nil {
			log.Printf("error during secret fetch: %v", err)
		} else if secret == nil {
			log.Printf("No secret found at %v", secretRef.Path)
		} else {
			secretMap[secretRef] = secret
		}
	}

	wrap(definition, secretMap)
}

func find(secretMap map[conf.GenericRef]*vaultapi.Secret, key string) (conf.GenericRef, *vaultapi.Secret) {
	for k, v := range secretMap {
		if ( k.Name == key ) {
			return k,v
		}
	}
}

func wrap(definition *conf.InjectionDefinition, secretMap map[conf.GenericRef]*vaultapi.Secret) {
	// Go spawn process with some env file
	//cmd := exec.Command("nc", "-l", "-p", "10000")
	cmd := exec.Command("bash", "-c", "set | grep ^DEMO")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()

	for envVarRef := range definition.Env {
		if envVarRef.Value != nil {
			env = append(env, fmt.Sprintf("%s=%s", envVarRef.Name, envVarRef.Value))
		} else envVarRef.ValueFrom != nil && envVarRef.ValueFrom.SecretKeyRef != nil {
			ref, secret := find(secretMap, envVarRef.ValueFrom.SecretKeyRef.Name)
			env = append(env, fmt.Sprintf("%s=%s", envVarRef.ValueFrom.SecretKeyRef.Name, secret.Data[envVarRef.ValueFrom.SecretKeyRef.Key].(string)))
		}
	}
	//

	cmd.Env = env
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	logger.GetLogger().Infof("Waiting for command to finish...")
	err = cmd.Wait()
	logger.GetLogger().Infof("Command finished with error: %v", err)

}
func test() *conf.InjectionDefinition {
	jsonBlob, err := ioutil.ReadFile("/opt/dev/workspace/vault-demo/examples/consul/kube-pod-decorator.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Converting yaml")
	definition := conf.GetInjectionDefinition(jsonBlob)
	//log.Printf("%v", definition)

	return &definition
}

// makeShutdownCh returns a channel that can be used for shutdown
// notifications for commands. This channel will send a message for every
// interrupt or SIGTERM received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		var shutdownInProgress = false
		for {
			<-signalCh
			if shutdownInProgress == false {
				logger.GetLogger().Debug("shutdown trigger")
				shutdownInProgress = true
			} else {
				logger.GetLogger().Fatal("Double shutdown triggered, killing the process")
			}
			resultCh <- struct{}{}
		}
	}()
	return resultCh
}

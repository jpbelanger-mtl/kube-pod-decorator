package wrapper

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/consul"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
)

func Wrap(definition *conf.InjectionDefinition, secretMap map[*conf.GenericRef]*vaultapi.Secret, consulMap map[*conf.GenericRef]*consulapi.KVPair, consulUtils *consul.ConsulUtils, shutdownChan <-chan struct{}) {
	env := os.Environ()

	// Import all defined environment variable into the process
	logger.GetLogger().Infof("Processing environment variables")
	envValues := getValues(secretMap, consulMap, definition.Env)
	for k, v := range envValues {
		logger.GetLogger().Infof("Processing env %v", k)
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create all requested files
	logger.GetLogger().Infof("Processing files")
	for _, envVarRef := range definition.Files {
		logger.GetLogger().Infof("Processing file %s", envVarRef.Name)
		kvPair, err := consulUtils.GetFile("files", envVarRef.Name)
		if err != nil || kvPair == nil {
			logger.GetLogger().Warningf("Could not get file %v", envVarRef.Name)
		} else {
			logger.GetLogger().Infof("Writing file %v to %v", envVarRef.Name, envVarRef.Destination)
			parentDir := path.Dir(envVarRef.Destination)
			if os.MkdirAll(parentDir, 0777) != nil {
				logger.GetLogger().Errorf("Unable to create directory '%v' for file '%v'", parentDir, envVarRef.Destination)
			}
			err := ioutil.WriteFile(envVarRef.Destination, []byte(kvPair.Value), 0644)
			if err != nil {
				logger.GetLogger().Errorf("Error while writing file %v", err)
			}
		}
	}

	// Create all requested templated files
	logger.GetLogger().Infof("Processing templates")
	for _, envVarRef := range definition.Templates {
		logger.GetLogger().Infof("Processing template %s", envVarRef.Name)
		kvPair, err := consulUtils.GetFile("templates", envVarRef.Name)
		if err != nil || kvPair == nil {
			logger.GetLogger().Warningf("Could not get template %v", envVarRef.Name)
		} else {
			templateValues := getValues(secretMap, consulMap, envVarRef.Env)

			t := template.New(envVarRef.Name)
			t, err = t.Parse(string(kvPair.Value))
			if err != nil {
				logger.GetLogger().Fatal(err)
			}
			logger.GetLogger().Infof("Writing templated file %v to %v", envVarRef.Name, envVarRef.Destination)
			parentDir := path.Dir(envVarRef.Destination)
			if os.MkdirAll(parentDir, 0777) != nil {
				logger.GetLogger().Errorf("Unable to create directory '%v' for templated file '%v'", parentDir, envVarRef.Destination)
			}
			f, err := os.Create(envVarRef.Destination)
			f.Chmod(0644)
			defer f.Close()
			if err != nil {
				logger.GetLogger().Errorf("File creation failure %v", err)
			}
			err = t.Execute(f, templateValues)
			if err != nil {
				logger.GetLogger().Errorf("Templating failure %v", err)
			}
		}
	}

	cmd := exec.Command(os.Args[1], os.Args[:2]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	logger.GetLogger().Infof("Starting command %v", os.Args[1])
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	logger.GetLogger().Infof("Waiting for command to finish...")
	MakeShutdownCh(cmd.Process)
	err = cmd.Wait()
	if err != nil {
		logger.GetLogger().Infof("Command finished with error: %v", err)
	}
}

func find(secretMap map[*conf.GenericRef]*vaultapi.Secret, consulMap map[*conf.GenericRef]*consulapi.KVPair, key string) (*conf.GenericRef, interface{}) {
	for k, v := range secretMap {
		if k.Name == key {
			return k, v
		}
	}
	for k, v := range consulMap {
		if k.Name == key {
			return k, v
		}
	}
	return nil, nil
}

func getValues(secretMap map[*conf.GenericRef]*vaultapi.Secret, consulMap map[*conf.GenericRef]*consulapi.KVPair, envs []*conf.EnvVar) map[string]string {
	values := make(map[string]string)
	for _, envVarRef := range envs {
		if len(envVarRef.Value) > 0 {
			//logger.GetLogger().Infof("Adding consul Env %s", envVarRef.Name)
			values[envVarRef.Name] = envVarRef.Value
		} else if envVarRef.ValueFrom != nil && envVarRef.ValueFrom.SecretKeyRef != nil {
			_, value := find(secretMap, consulMap, envVarRef.ValueFrom.SecretKeyRef.Name)
			if value != nil {
				//logger.GetLogger().Infof("Adding secret Env %s", envVarRef.Name)
				secret := value.(*vaultapi.Secret)
				values[envVarRef.Name] = secret.Data[envVarRef.ValueFrom.SecretKeyRef.Key].(string)
			} else {
				logger.GetLogger().Warningf("Could not find secret for %s with key %s", envVarRef.Name, envVarRef.ValueFrom.SecretKeyRef.Name)
			}
		} else if envVarRef.ValueFrom != nil && envVarRef.ValueFrom.Consul != nil {
			ref, value := find(secretMap, consulMap, envVarRef.ValueFrom.Consul.Name)
			if value != nil {
				kvPair := value.(*consulapi.KVPair)
				if ref.Type == "json" {
					var jsonObj map[string]interface{}
					err := json.Unmarshal([]byte(kvPair.Value), &jsonObj)
					if err != nil {
						logger.GetLogger().Errorf("Error unmarshalling json for %v : %v", envVarRef.Name, kvPair.Value)
					} else {
						values[envVarRef.Name] = jsonObj[envVarRef.ValueFrom.Consul.Key].(string)
					}
				} else {
					values[envVarRef.Name] = string(kvPair.Value)
				}
			} else {
				logger.GetLogger().Warningf("Could not find consul KV for %s with key %s", envVarRef.Name, envVarRef.ValueFrom.Consul.Name)
			}
		} else {
			logger.GetLogger().Warningf("Nothing found to do with %s", envVarRef.Name)
		}
	}
	return values
}

// makeShutdownCh returns a channel that can be used for shutdown
// notifications for commands. This channel will send a message for every
// interrupt or SIGTERM received.
func MakeShutdownCh(process *os.Process) <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		var shutdownInProgress = false
		for {
			signal := <-signalCh
			if shutdownInProgress == false {
				logger.GetLogger().Debug("shutdown trigger")
				if process != nil {
					logger.GetLogger().Debugf("Trigerring shutdown on wrapped process with signel %v", signal)
					process.Signal(signal)
				}
				shutdownInProgress = true
			} else {
				logger.GetLogger().Fatal("Double shutdown triggered")
				if process != nil {
					process.Kill()
				}
			}
			resultCh <- struct{}{}
		}
	}()
	return resultCh
}

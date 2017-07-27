package consul

import (
	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/jpbelanger-mtl/kube-pod-decorator/conf"
	"github.com/jpbelanger-mtl/kube-pod-decorator/logger"
)

type ConsulUtils struct {
	config conf.Specification
	client *consul.Client
}

func NewConsulUtils(
	config conf.Specification,
	token string,
) *ConsulUtils {
	cu := ConsulUtils{
		config: config,
	}
	cu.init(token)
	return &cu
}

func (cu *ConsulUtils) init(token string) {
	config := consul.Config{Token: token}
	client, err := consul.NewClient(&config)
	if err != nil {
		logger.GetLogger().Fatalf("Error while creating consul client: %v", err)
	}
	cu.client = client
}

func (cu *ConsulUtils) GetPodConfig() (*consul.KVPair, error) {
	path := fmt.Sprintf("%v/%v/config", cu.config.ConsulConfigRoot, cu.config.ApplicationName)
	logger.GetLogger().Infof("Fetching pod's config at %v", path)
	kv, _, err := cu.client.KV().Get(path, &consul.QueryOptions{})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func (cu *ConsulUtils) GetFile(folder string, filename string) (*consul.KVPair, error) {
	path := fmt.Sprintf("%v/%v/%v/%v", cu.config.ConsulConfigRoot, cu.config.ApplicationName, folder, filename)
	logger.GetLogger().Infof("Fetching file at %v", path)
	kv, _, err := cu.client.KV().Get(path, &consul.QueryOptions{})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func (cu *ConsulUtils) GetValue(path string) (*consul.KVPair, error) {
	logger.GetLogger().Infof("Fetching consul value at %v", path)
	kv, _, err := cu.client.KV().Get(path, &consul.QueryOptions{})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

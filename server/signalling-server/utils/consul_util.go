package utils

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/consul/api"
)

func discoverService(serviceName string) (string, error) {
	config := api.DefaultConfig()
	client, err := api.NewClient(config)
	if err != nil {
		return "", err
	}

	services, err := client.Agent().Services()
	if err != nil {
		return "", err
	}

	for _, service := range services {
		if service.Service == serviceName {
			return service.Address + ":" + strconv.Itoa(service.Port), nil
		}
	}

	return "", fmt.Errorf("service %s not found", serviceName)
}

func fetchConfig(client *api.Client, key string) string {
	kv, _, err := client.KV().Get("service/config/"+key, nil)
	if err != nil || kv == nil {
		log.Fatal("Error retreiving KV from Consul:", err)
	}
	return string(kv.Value)
}

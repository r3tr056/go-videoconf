package servicediscovery

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

type ConsulServiceRegistery struct {
	client          *api.Client
	serviceName     string
	servicePort     int
	serviceAddress  string
	healthCheckPath string
}

func NewConsulRegistry(consulAddress string, serviceName string, servicePort int, serviceAddress string, healthCheckPath string) (*ConsulServiceRegistery, error) {
	config := api.DefaultConfig()
	config.Address = consulAddress
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	return &ConsulServiceRegistery{
		client:          client,
		serviceName:     serviceName,
		servicePort:     servicePort,
		serviceAddress:  serviceAddress,
		healthCheckPath: healthCheckPath,
	}, nil
}

func (r *ConsulServiceRegistery) RegisterService() error {
	address := r.serviceAddress
	if address == "" {
		// determine outbound host IP if not explicitly provided
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return fmt.Errorf("failed to get interface addresses: %w", err)
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				address = ipnet.IP.String()
				break
			}
		}
		if address == "" {
			address = "localhost"
		}
	}

	registration := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s-%d", r.serviceName, address, r.servicePort),
		Name:    r.serviceName,
		Port:    r.servicePort,
		Address: address,
		Check: &api.AgentServiceCheck{
			Interval:                       "10s",
			Timeout:                        "5s",
			HTTP:                           fmt.Sprintf("http://%s:%d%s", address, r.servicePort, r.healthCheckPath),
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	err := r.client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("failed to register service with Consul: %w", err)
	}
	logrus.Infof("Registered service '%s' with Consul at %s:%d", r.serviceName, address, r.servicePort)
	return nil
}

func (r *ConsulServiceRegistery) DeregisterService() error {
	serviceID := fmt.Sprintf("%s-%s-%d", r.serviceName, r.serviceAddress, r.servicePort)
	err := r.client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		return fmt.Errorf("failed to deregister service from Consul: %w", err)
	}
	logrus.Infof("Deregistered service '%s' from Consul", r.serviceName)
	return nil
}

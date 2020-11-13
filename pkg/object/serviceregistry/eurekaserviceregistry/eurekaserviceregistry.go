package eurekaserviceregistry

import (
	"fmt"
	"sync"
	"time"

	"github.com/megaease/easegateway/pkg/logger"
	"github.com/megaease/easegateway/pkg/object/serviceregistry"
	"github.com/megaease/easegateway/pkg/scheduler"

	"github.com/ArthurHlt/go-eureka-client/eureka"
)

const (
	// Kind is EurekaServiceRegistry kind.
	Kind = "EurekaServiceRegistry"
)

func init() {
	scheduler.Register(&scheduler.ObjectRecord{
		Kind:              Kind,
		DefaultSpecFunc:   DefaultSpec,
		NewFunc:           New,
		DependObjectKinds: nil,
	})
}

type (
	// EurekaServiceRegistry is Object EurekaServiceRegistry.
	EurekaServiceRegistry struct {
		spec *Spec

		clientMutex sync.RWMutex
		client      *eureka.Client

		statusMutex sync.Mutex
		serversNum  map[string]int

		done chan struct{}
	}

	// Spec describes the EurekaServiceRegistry.
	Spec struct {
		scheduler.ObjectMeta `yaml:",inline"`

		Endpoints    []string `yaml:"endpoints" jsonschema:"required,uniqueItems=true"`
		SyncInterval string   `yaml:"syncInterval" jsonschema:"required,format=duration"`
	}

	// Status is the status of EurekaServiceRegistry.
	Status struct {
		Timestamp  int64          `yaml:"timestamp"`
		Health     string         `yaml:"health"`
		ServersNum map[string]int `yaml:"serversNum"`
	}
)

// DefaultSpec returns EurekaServiceRegistry default spec.
func DefaultSpec() *Spec {
	return &Spec{
		Endpoints:    []string{"http://127.0.0.1:8761/eureka"},
		SyncInterval: "10s",
	}
}

// Validate validates Spec.
func (spec Spec) Validate() error {
	return nil
}

// New creates an EurekaServiceRegistry.
func New(spec *Spec, prev *EurekaServiceRegistry, handlers *sync.Map) *EurekaServiceRegistry {
	esr := &EurekaServiceRegistry{
		spec:       spec,
		serversNum: map[string]int{},
		done:       make(chan struct{}),
	}
	if prev != nil {
		prev.Close()
	}

	_, err := esr.getClient()
	if err != nil {
		logger.Errorf("%s get consul client failed: %v", spec.Name, err)
	}

	go esr.run()

	return esr
}

func (esr *EurekaServiceRegistry) getClient() (*eureka.Client, error) {
	esr.clientMutex.RLock()
	if esr.client != nil {
		client := esr.client
		esr.clientMutex.RUnlock()
		return client, nil
	}
	esr.clientMutex.RUnlock()

	return esr.buildClient()
}

func (esr *EurekaServiceRegistry) buildClient() (*eureka.Client, error) {
	esr.clientMutex.Lock()
	defer esr.clientMutex.Unlock()

	// DCL
	if esr.client != nil {
		return esr.client, nil
	}

	client := eureka.NewClient(esr.spec.Endpoints)

	esr.client = client

	return client, nil
}

func (esr *EurekaServiceRegistry) closeClient() {
	esr.clientMutex.Lock()
	defer esr.clientMutex.Unlock()

	if esr.client == nil {
		return
	}

	esr.client = nil
}

func (esr *EurekaServiceRegistry) run() {
	syncInterval, err := time.ParseDuration(esr.spec.SyncInterval)
	if err != nil {
		logger.Errorf("BUG: parse duration %s failed: %v",
			esr.spec.SyncInterval, err)
		return
	}

	esr.update()

	for {
		select {
		case <-esr.done:
			return
		case <-time.After(syncInterval):
			esr.update()
		}
	}
}

func (esr *EurekaServiceRegistry) update() {
	client, err := esr.getClient()
	if err != nil {
		logger.Errorf("%s get consul client failed: %v",
			esr.spec.Name, err)
		return
	}

	apps, err := client.GetApplications()
	if err != nil {
		logger.Errorf("%s get services failed: %v",
			esr.spec.Name, err)
		return
	}

	servers := []*serviceregistry.Server{}
	serversNum := map[string]int{}
	for _, app := range apps.Applications {
		for _, instance := range app.Instances {
			baseServer := serviceregistry.Server{
				ServiceName: app.Name,
				Hostname:    instance.HostName,
				HostIP:      instance.IpAddr,
				Port:        int16(instance.Port.Port),
			}
			if instance.Port != nil && instance.Port.Enabled {
				server := baseServer

				fmt.Printf("server: %+v\n", server)
				servers = append(servers, &server)
				serversNum[app.Name]++
			}

			if instance.SecurePort != nil && instance.SecurePort.Enabled {
				server := baseServer
				server.Scheme = "https"
				fmt.Printf("server: %+v\n", server)
				servers = append(servers, &server)
				serversNum[app.Name]++
			}
		}
	}

	serviceregistry.Global.ReplaceServers(esr.spec.Name, servers)

	esr.statusMutex.Lock()
	esr.serversNum = serversNum
	esr.statusMutex.Unlock()
}

// Status returns status of EurekaServiceRegister.
func (esr *EurekaServiceRegistry) Status() *Status {
	s := &Status{}

	_, err := esr.getClient()
	if err != nil {
		s.Health = err.Error()
	} else {
		s.Health = "ready"
	}

	esr.statusMutex.Lock()
	serversNum := esr.serversNum
	esr.statusMutex.Unlock()

	s.ServersNum = serversNum

	return s
}

// Close closes EurekaServiceRegistry.
func (esr *EurekaServiceRegistry) Close() {
	esr.closeClient()
	close(esr.done)

	serviceregistry.Global.CloseRegistry(esr.spec.Name)
}
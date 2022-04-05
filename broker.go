package goservice

import (
	"time"

	"github.com/bep/debounce"
)

// BROKER
type BrokerConfig struct {
	NodeId            string
	TransporterConfig TransporterConfig
	Logger            string
	Matrics           string
	Trace             string
	DiscoveryConfig   DiscoveryConfig
	RequestTimeOut    int
}

type Broker struct {
	Config   BrokerConfig
	Services []*Service
	Started  func(*Context)
	Stoped   func(*Context)
}

var broker Broker

var debouncedEmitInfo func(f func())

func Init() {
	broker = Broker{
		Config: BrokerConfig{
			NodeId: "Node-1",
			DiscoveryConfig: DiscoveryConfig{
				DiscoveryType: DiscoveryTypeRedis,
				Config: DiscoveryRedisConfig{
					Port: 6379,
					Host: "127.0.0.1",
				},
			},
			TransporterConfig: TransporterConfig{
				TransporterType: TransporterTypeRedis,
				Config: TransporterRedisConfig{
					Port: 6379,
					Host: "127.0.0.1",
				},
			},
			RequestTimeOut: 30000,
		},
	}
	initDiscovery()
	initTransporter()
	debouncedEmitInfo = debounce.New(1000 * time.Millisecond)
}

func LoadService(service Service) {
	// add service to list
	broker.Services = append(broker.Services, &service)

	// add service to registry
	var registryActions []RegistryAction
	for _, a := range service.Actions {
		registryActions = append(registryActions, RegistryAction{
			Name:   a.Name,
			Params: a.Params,
		})
	}
	var registryEvents []RegistryEvent
	for _, e := range service.Events {
		registryEvents = append(registryEvents, RegistryEvent{
			Name:   e.Name,
			Params: e.Params,
		})
	}
	registryServices = append(registryServices, RegistryService{
		Node:    registryNode,
		Name:    service.Name,
		Actions: registryActions,
		Events:  registryEvents,
	})

	// emit info service
	debouncedEmitInfo(startDiscovery)
	// handle logic service

	// service lifecycle
	// started
	context := Context{
		RequestId:    "",
		Params:       map[string]interface{}{},
		Meta:         map[string]interface{}{},
		FromService:  "",
		FromNode:     "",
		CallingLevel: 1,
	}
	context.Call = func(action string, params interface{}, meta interface{}) (interface{}, error) {
		return callAction(context, action, params, meta)
	}

	service.Started(&context)
	// actions handle
	for _, a := range service.Actions {
		listenActionCall(service.Name, a)
	}

	// events handle
}

func Hold() {
	select {}
}

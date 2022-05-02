package goservice

type RegistryNode struct {
	NodeId     string   `json:"node_id" mapstructure:"node_id"`
	IP         []string `json:"ip" mapstructure:"ip"`
	LastActive int      `json:"last_active" mapstructure:"last_active"`
}
type RegistryAction struct {
	Name   string      `json:"name" mapstructure:"name"`
	Rest   Rest        `json:"rest" mapstructure:"rest"`
	Params interface{} `json:"params" mapstructure:"params"`
}
type RegistryEvent struct {
	Name   string      `json:"name" mapstructure:"name"`
	Params interface{} `json:"params" mapstructure:"params"`
}
type RegistryService struct {
	Node    RegistryNode     `json:"node" mapstructure:"node"`
	Name    string           `json:"name" mapstructure:"name"`
	Actions []RegistryAction `json:"actions" mapstructure:"actions"`
	Events  []RegistryEvent  `json:"events" mapstructure:"events"`
}

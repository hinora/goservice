package goservice

type RegistryNode struct {
	NodeId string
	IP     []string
}
type RegistryAction struct {
	Name   string
	Params interface{}
}
type RegistryEvent struct {
	Name   string
	Params interface{}
}
type RegistryService struct {
	Node    RegistryNode
	Name    string
	Actions []RegistryAction
	Events  []RegistryEvent
}

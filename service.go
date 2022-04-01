package goservice

type Call func(action string, params interface{}, meta interface{}) (interface{}, error)
type Context struct {
	RequestId   string
	Params      interface{}
	Meta        interface{}
	FromService string
	FromNode    string
	Call        Call
}
type Action struct {
	Name   string
	Params interface{}
	Handle func(*Context) (interface{}, error)
}
type Event struct {
	Name   string
	Params interface{}
	Handle func(*Context)
}
type Service struct {
	Name    string
	Actions []Action
	Events  []Event
	Started func(*Context)
	Stoped  func(*Context)
}

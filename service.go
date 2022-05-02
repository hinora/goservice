package goservice

type Call func(action string, params interface{}, meta interface{}) (interface{}, error)
type Context struct {
	RequestId         string
	TraceParentId     string
	TraceParentRootId string
	ResponseId        string
	Params            interface{}
	Meta              interface{}
	FromService       string
	FromAction        string
	FromEvent         string
	FromNode          string
	CallingLevel      int
	Call              Call
}

type Method int

const (
	GET Method = iota + 1
	POST
	PUT
	DELETE
	PATCH
	HEAD
	OPTIONS
)

type Rest struct {
	Method Method `json:"method" mapstructure:"method"`
	Path   string `json:"path" mapstructure:"path"`
}
type Action struct {
	Name   string
	Params interface{}
	Rest   Rest
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

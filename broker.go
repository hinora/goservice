package goservice

// BROKER
type BrokerConfig struct {
	NodeId      string
	Transporter string
	Logger      string
	Matrics     string
	Trace       string
}

type Broker struct {
	Config   BrokerConfig
	Services []Service
	Started  func(*Context)
	Stoped   func(*Context)
}

var broker Broker

func Init() {

}

func Hold() {
	select {}
}

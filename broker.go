package goservice

// BROKER
type BrokerConfig struct {
	NodeId      string
	Transporter string
	Logger      string
	Matrics     string
	Trace       string
}

var broker BrokerConfig = BrokerConfig{
	NodeId: "Test",
}

func Hold() {
	select {}
}

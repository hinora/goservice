package goservice

import (
	"reflect"
)

type EventBusData struct {
	Channel string
	Handle  []reflect.Value
}

type EventBus struct {
	data []EventBusData
}

func (eb *EventBus) Subscribe(channel string, fn interface{}) {
	check := false
	for _, e := range eb.data {
		if e.Channel == channel {
			e.Handle = append(e.Handle, reflect.ValueOf(fn))
			check = true
		}
	}
	if !check {
		eb.data = append(eb.data, EventBusData{
			Channel: channel,
			Handle: []reflect.Value{
				reflect.ValueOf(fn),
			},
		})
	}

}

func (eb *EventBus) Publish(channel string, data interface{}) {
	for _, e := range eb.data {
		if e.Channel == channel {
			for _, h := range e.Handle {
				go func() {
					h.Call([]reflect.Value{
						reflect.ValueOf(data),
					})
				}()
			}
		}
	}
}
func (eb *EventBus) UnSubscribe(channel string) {
	index := -1
	for i, e := range eb.data {
		if e.Channel == channel {
			index = i
		}
	}
	if index != -1 {
		eb.data[index] = eb.data[len(eb.data)-1]
		eb.data = eb.data[:len(eb.data)-1]
	}
}

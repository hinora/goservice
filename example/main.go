package main

import (
	"fmt"

	"github.com/hinora/goservice"
)

func main() {
	goservice.Init()
	goservice.LoadService(goservice.Service{
		Name: "math",
		Started: func(ctx *goservice.Context) {
			fmt.Println("service test started")
		},
		Actions: []goservice.Action{
			{
				Name:   "plus",
				Params: map[string]interface{}{},
				Handle: func(context *goservice.Context) (interface{}, error) {
					fmt.Println("Handle action plus")
					return nil, nil
				},
			},
		},
	})
	goservice.Hold()
}

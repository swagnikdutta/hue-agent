package main

import (
	"sync"

	"github.com/openhue/openhue-go"
)

type HueAgent struct {
	Wg   *sync.WaitGroup
	home *openhue.Home
}

type LightStateRequest struct {
	On         bool    `json:"on,omitempty"`
	Mirek      int     `json:"mirek,omitempty"`
	Brightness float32 `json:"brightness,omitempty"`
}

type Response struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

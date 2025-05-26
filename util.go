package main

import "encoding/json"

func prepareHTTPResponse(code int, msg string, data any) []byte {
	resp := Response{
		Code:    code,
		Message: msg,
		Data:    data,
	}
	r, _ := json.Marshal(resp)
	return r
}

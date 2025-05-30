package entities

import "net/http"

type ProxyResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Err        error
}

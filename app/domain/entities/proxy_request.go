package entities

import "net/http"

type ProxyRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
	Reply   chan ProxyResponse
}

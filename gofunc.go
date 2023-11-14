package gofunc

import (
	"github.com/ottstack/gofunc/internal/serve"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/valyala/fasthttp"
)

var globalServer = serve.NewServer()

func Get(path string, function interface{}) {
	handle("GET", path, function)
}
func Post(path string, function interface{}) {
	handle("POST", path, function)
}
func Delete(path string, function interface{}) {
	handle("DELETE", path, function)
}
func Put(path string, function interface{}) {
	handle("PUT", path, function)
}
func Stream(path string, function interface{}) {
	handle("STREAM", path, function)
}

func Use(m middleware.Middleware) *serve.Server {
	return globalServer.Use(m)
}

func HandleHTTP(method, path string, f func(*fasthttp.RequestCtx)) {
	err := globalServer.Handle(method, path, f)
	if err != nil {
		panic(err)
	}
}

// Serve ...
func Serve() {
	if err := globalServer.Serve(); err != nil {
		panic(err)
	}
}

func handle(method, path string, function interface{}) {
	err := globalServer.Handle(method, path, function)
	if err != nil {
		panic(err)
	}
}

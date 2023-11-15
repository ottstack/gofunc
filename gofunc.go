package gofunc

import (
	"github.com/ottstack/gofunc/internal/serve"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/valyala/fasthttp"
)

var globalServer = serve.NewServer()

func Get(path string, function interface{}, opts ...applyFunc) {
	handle("GET", path, function, opts...)
}
func Post(path string, function interface{}, opts ...applyFunc) {
	handle("POST", path, function, opts...)
}
func Delete(path string, function interface{}, opts ...applyFunc) {
	handle("DELETE", path, function, opts...)
}
func Put(path string, function interface{}, opts ...applyFunc) {
	handle("PUT", path, function, opts...)
}
func Stream(path string, function interface{}, opts ...applyFunc) {
	handle("STREAM", path, function, opts...)
}

func Use(m middleware.Middleware) *serve.Server {
	return globalServer.Use(m)
}

func HandleHTTP(method, path string, f func(*fasthttp.RequestCtx)) {
	err := globalServer.Handle(method, path, f, "", nil)
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

func handle(method, path string, function interface{}, opts ...applyFunc) {
	opt := &funcOption{}
	for _, op := range opts {
		op(opt)
	}
	if opt.tags == nil {
		opt.tags = []string{"Default"}
	}
	err := globalServer.Handle(method, path, function, opt.summary, opt.tags)
	if err != nil {
		panic(err)
	}
}

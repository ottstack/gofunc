package gofunc

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/ottstack/gofunc/internal/serve"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/valyala/fasthttp"
)

var globalServer = serve.NewServer()

func New(name string) *Router {
	return &Router{name: name}
}

type Router struct {
	name string
}

func (r *Router) Get(path string, function interface{}) *Router {
	r.handle("GET", path, function)
	return r
}
func (r *Router) Post(path string, function interface{}) *Router {
	r.handle("POST", path, function)
	return r
}
func (r *Router) Delete(path string, function interface{}) *Router {
	r.handle("DELETE", path, function)
	return r
}
func (r *Router) Put(path string, function interface{}) *Router {
	r.handle("PUT", path, function)
	return r
}
func (r *Router) Stream(path string, function interface{}) *Router {
	r.handle("STREAM", path, function)
	return r
}

func (r *Router) handle(method, path string, function interface{}) {
	name := getFunctionName(function)
	if idx := strings.IndexRune(name, '.'); idx >= 0 {
		name = name[idx+1:]
	}
	err := globalServer.Handle(method, path, function, name, r.name)
	if err != nil {
		panic(err)
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func Use(m middleware.Middleware) *serve.Server {
	return globalServer.Use(m)
}

func HandleHTTP(method, path string, f func(*fasthttp.RequestCtx)) {
	err := globalServer.Handle(method, path, f, "", "")
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

package gofunc

import (
	"log"

	"github.com/ottstack/gofunc/internal/serve"
	"github.com/ottstack/gofunc/pkg/middleware"
)

var globalServer = serve.NewServer()

func Get(path string, function interface{}) {
	handle("GET", path, function)
}

func Use(m middleware.Middleware) *serve.Server {
	return globalServer.Use(m)
}

// Serve ...
func Serve() {
	if err := globalServer.Serve(); err != nil {
		log.Fatal(err)
	}
}

func handle(method, path string, function interface{}) {
	err := globalServer.Handle(method, path, function)
	if err != nil {
		log.Fatal(err)
	}
}

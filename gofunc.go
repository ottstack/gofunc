package gofunc

import (
	"log"

	"github.com/ottstack/gofunc/internal/serve"
	"github.com/ottstack/gofunc/pkg/middleware"
)

var globalServer = serve.NewServer()

// Handle ...
func Handle(handlers ...interface{}) {
	err := globalServer.Handle(handlers...)
	if err != nil {
		log.Fatal(err)
	}
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

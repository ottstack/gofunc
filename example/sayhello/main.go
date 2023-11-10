package main

import (
	"context"
	"fmt"

	"github.com/ottstack/gofunc"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/ottstack/gofunc/pkg/websocket"
)

type Request struct {
	Name string `schema:"name" validate:"required"` // decode from query by github.com/gorilla/schema
}

type Response struct {
	Reply string `json:"reply"`
}

func HelloFunc(ctx context.Context, req *Request, rsp *Response) error {
	rsp.Reply = "Hello " + req.Name
	return nil
}

func Stream(ctx context.Context, req websocket.RecvStream, rsp websocket.SendStream) error {
	ct := 0
	for {
		msg, err := req.Recv()
		if err != nil {
			return err
		}
		fmt.Println("recv", string(msg))
		ct++
		if err := rsp.Send([]byte(fmt.Sprintf("hello %s %d times", string(msg), ct))); err != nil {
			return err
		}
		fmt.Println("send", string(msg))
		if ct > 2 {
			return nil
		}
	}
}

func main() {

	// curl '127.0.0.1:9001/api/hello?name=bob'
	// curl '127.0.0.1:9001/api/hello' -d '{"name":"tom"}'
	// websocket: 127.0.0.1:9001/api/hello-ws
	gofunc.Get("/api/hello", HelloFunc)
	gofunc.Post("/api/hello", HelloFunc)
	gofunc.Stream("/api/hello-ws", Stream)

	gofunc.Use(middleware.Recover).Use(middleware.Validator)
	gofunc.Serve()
}

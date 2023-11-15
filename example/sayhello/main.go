package main

import (
	"context"
	"fmt"

	"github.com/ottstack/gofunc"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/ottstack/gofunc/pkg/websocket"
	"github.com/valyala/fasthttp"
)

type Request struct {
	Name string `json:"name" schema:"name" validate:"required" comment:"Required Name"`
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
	gofunc.Get("/api/hello", HelloFunc, gofunc.WithSummary("Get Example"))

	// curl -X PUT '127.0.0.1:9001/api/hello' -d '{"name":"tom"}'
	gofunc.Put("/api/hello", HelloFunc, gofunc.WithSummary("Put Example"))

	// websocket: 127.0.0.1:9001/api/hello-ws
	gofunc.Stream("/api/hello-ws", Stream, gofunc.WithSummary("Websocket Example"))

	// curl '127.0.0.1:9001/api/hello/2'
	gofunc.HandleHTTP("GET", "/api/hello/2", func(rc *fasthttp.RequestCtx) {
		rc.Response.BodyWriter().Write([]byte("HELLO FAST HTTP"))
	})

	gofunc.Use(middleware.Recover).Use(middleware.Validator)
	gofunc.Serve()
}

package main

import (
	"context"
	"fmt"

	"github.com/ottstack/gofunc"
	"github.com/ottstack/gofunc/pkg/websocket"
)

type GetRequest struct {
	Name string `schema:"name" validate:"required"` // decode from query by github.com/gorilla/schema
}

type PostRequest struct {
	Name string `json:"name" validate:"required"` // decode from json body by github.com/goccy/go-json
}

type Response struct {
	Reply string `json:"reply"`
}

type StreamRequest struct {
	Name string `validate:"required"`
}

func Get(ctx context.Context, req *GetRequest, rsp *Response) error {
	rsp.Reply = "Get by " + req.Name
	return nil
}

func Post(ctx context.Context, req *PostRequest, rsp *Response) error {
	rsp.Reply = "Post by " + req.Name
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

// val := reflect.ValueOf(abcGet)

// ctx := context.Background()
// req := &GetRequest{Name: "abc"}
// rsp := &Response{}
// args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req), reflect.ValueOf(rsp)}
// fmt.Println(val.Call(args))
// fmt.Println("rsp", rsp.Reply)
// return

func main() {

	// GET /api/hello?name=bob
	// POST /api/hello -d '{"name":"bob"}'
	// GET /api/hello-ws?name=bob (websocket)
	gofunc.Get("/api/hello", Get)
	// gofunc.Post("/api/hello", Post)
	// gofunc.Stream("/api/hello", Stream)

	// gofunc.Use(middleware.Recover).Use(middleware.Validator)
	gofunc.Serve()
}

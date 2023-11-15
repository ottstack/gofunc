package serve

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strings"

	"github.com/fasthttp/websocket"
	"github.com/kelseyhightower/envconfig"
	"github.com/ottstack/gofunc/pkg/ecode"
	"github.com/ottstack/gofunc/pkg/middleware"
	"github.com/valyala/fasthttp"
	"go.uber.org/automaxprocs/maxprocs"
)

var allowMethod = map[string]bool{
	"GET":    true,
	"POST":   true,
	"DELETE": true,
	"PUT":    true,
	"STREAM": true,
}

type Server struct {
	methods       map[string]methodFactory
	streamMethods map[string]bool
	api           *openapi
	middlewares   []middleware.Middleware
	ctx           context.Context
	cancelFunc    context.CancelFunc
	addr          string
	swaggerPath   string
	pathMapping   map[string]string
	apiContent    []byte

	rawHandler map[string]func(*fasthttp.RequestCtx)

	crossDomain bool
}

type serveConfig struct {
	Addr        string
	SwaggerPath string
}

type methodFactory func() (middleware.MethodFunc, interface{}, interface{})
type methodInfo struct {
	method      interface{}
	operationId string

	tags    []string
	summary string

	httpMethod  string
	factory     methodFactory
	reqType     reflect.Type
	rspType     reflect.Type
	path        string
	isWebsocket bool
}

func NewServer() *Server {
	cfg := &serveConfig{
		Addr:        "127.0.0.1:9001",
		SwaggerPath: "/",
	}
	err := envconfig.Process("serve", cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	sv := &Server{
		swaggerPath:   cfg.SwaggerPath,
		addr:          cfg.Addr,
		ctx:           ctx,
		cancelFunc:    cancelFunc,
		methods:       make(map[string]methodFactory),
		streamMethods: make(map[string]bool),
		crossDomain:   false,
		rawHandler:    make(map[string]func(*fasthttp.RequestCtx)),
	}
	sv.api = newOpenapi(cfg.SwaggerPath)
	sv.api.parseType("", rspFieldTag, reflect.TypeOf(&ecode.APIError{}))
	return sv
}

func (s *Server) Handle(method, path string, function interface{}, summary string, tags []string) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !allowMethod[method] {
		return fmt.Errorf("http method %s is unsupported", method)
	}
	checkMethod := method
	if method == "STREAM" {
		checkMethod = "GET"
	}
	methodPath := checkMethod + "_" + path
	if _, ok := s.methods[methodPath]; ok {
		return fmt.Errorf("%s %s already registered", method, path)
	}
	if _, ok := s.rawHandler[methodPath]; ok {
		return fmt.Errorf("%s %s already registered for http handler", method, path)
	}
	if vv, ok := function.(func(*fasthttp.RequestCtx)); ok {
		s.rawHandler[methodPath] = vv
		return nil
	}
	info := &methodInfo{
		httpMethod:  method,
		path:        path,
		summary:     summary,
		tags:        tags,
		method:      function,
		operationId: method + strings.ReplaceAll(path, "/", "_"),
	}
	if err := parseMethods(info); err != nil {
		return err
	}

	if info.httpMethod == "STREAM" {
		info.httpMethod = "GET"
		s.streamMethods[path] = true
	}

	s.methods[methodPath] = info.factory

	s.api.addMethod(info)
	return nil
}

func (s *Server) Use(m middleware.Middleware) *Server {
	s.middlewares = append(s.middlewares, m)
	return s
}

func (s *Server) Serve() error {
	defer s.cancelFunc()
	// maxprocs
	maxprocs.Set(maxprocs.Logger(func(s string, args ...interface{}) {
		log.Printf(s, args...)
	}))

	showAddr := s.addr
	addrInfo := strings.SplitN(s.addr, ":", 2)
	if addrInfo[0] == "" || addrInfo[0] == "0" || addrInfo[0] == "0.0.0.0" {
		showAddr = "localhost:" + addrInfo[1]
	}
	log.Println("Serving API on http://" + showAddr + s.swaggerPath)
	s.apiContent = s.api.getOpenAPIV3()
	return fasthttp.ListenAndServe(s.addr, s.serve)
}

// serve serve as http handler
func (s *Server) serve(fastReq *fasthttp.RequestCtx) {
	// serve openapi
	path := string(fastReq.Path())
	method := strings.ToUpper(string(fastReq.Method()))
	if path == s.swaggerPath+"api.json" {
		fastReq.Write(s.apiContent)
		return
	}
	if path == s.swaggerPath {
		fastReq.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
		fastReq.Write(s.api.getSwaggerHTML())
		return
	}
	if path == s.swaggerPath+"doc" {
		fastReq.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
		fastReq.Write(s.api.getDocHTML())
		return
	}

	methodPath := method + "_" + path
	hd, ok := s.rawHandler[methodPath]
	if ok {
		hd(fastReq)
		return
	}

	if s.crossDomain {
		referer := string(fastReq.Referer())
		if u, _ := url.Parse(referer); u != nil {
			fastReq.Response.Header.Set("Access-Control-Allow-Origin", fmt.Sprintf("%s://%s", u.Scheme, u.Host))
		} else {
			fastReq.Response.Header.Set("Access-Control-Allow-Origin", "*")
		}
		fastReq.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		fastReq.Response.Header.Set("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		fastReq.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,DELETE,PUT")
		if method == "OPTIONS" {
			return
		}
	}

	// path to func
	factory, ok := s.methods[methodPath]
	if !ok {
		writeErrResponse(fastReq, &ecode.APIError{Code: 404, Message: fmt.Sprintf("Request %s %s not found", method, path)})
		return
	}
	realMethod, req, rsp := factory()

	var reqBody []byte
	decoder := jsonDecoder
	isWebsocket := s.streamMethods[path]
	var stream *streamImp

	doCallFunc := func() {
		if len(reqBody) > 0 {
			if err := decoder(reqBody, req); err != nil {
				writeErrResponse(fastReq, &ecode.APIError{Code: 400, Message: "Decode request body failed: " + err.Error()})
				return
			}
		}

		ctx := context.Background()

		// Middleware
		for i := range s.middlewares {
			mware := s.middlewares[len(s.middlewares)-i-1]
			realMethod = func(mm middleware.MethodFunc) middleware.MethodFunc {
				return func(ctx context.Context, req, rsp interface{}) error {
					return mware(ctx, fastReq, mm, req, rsp)
				}
			}(realMethod)
		}
		err := realMethod(ctx, req, rsp)
		if isWebsocket {
			return
		}
		if err != nil {
			writeErrResponse(fastReq, err)
			return
		}

		fastReq.Response.Header.Set("Content-Type", "application/json")
		reqBody, err = encoder(rsp)
		if err != nil {
			writeErrResponse(fastReq, fmt.Errorf("marshal rsp error: %v", err))
			return
		}
		fastReq.Write(reqBody)
	}

	if isWebsocket {
		err := upgrader.Upgrade(fastReq, func(conn *websocket.Conn) {
			stream = rsp.(*streamImp)
			stream.conn = conn
			defer stream.close()
			doCallFunc()
		})
		if err != nil {
			log.Println("Upgrade websocket error: ", err.Error())
		}
		return
	} else if method == "POST" || method == "PUT" {
		reqBody = fastReq.PostBody()
	} else {
		reqBody = fastReq.URI().QueryString()
		decoder = queryDecoder
	}
	doCallFunc()
}

func (s *Server) PathMapping(m map[string]string) *Server {
	s.pathMapping = m
	return s
}

func parseMethods(m *methodInfo) error {
	method := reflect.TypeOf(m.method)
	if method.NumIn() != 3 {
		return fmt.Errorf("the number of argment in %s should be 3", m.path)
	}
	if method.NumOut() != 1 {
		return fmt.Errorf("the number of return value in %s should be 1", m.path)
	}

	ctx := method.In(0)
	req := method.In(1)
	rsp := method.In(2)

	if ctx.PkgPath() != "context" || ctx.Name() != "Context" {
		return fmt.Errorf("first argment in %s should be context.Context", m.path)
	}

	if m.httpMethod == "STREAM" {
		if req.Kind() != reflect.Interface || req.Name() != "RecvStream" {
			return fmt.Errorf("the type of third argment in %s should be websocket.RecvStream", m.path)
		}
		if rsp.Kind() != reflect.Interface || rsp.Name() != "SendStream" {
			return fmt.Errorf("the type of third argment in %s should be websocket.SendStream", m.path)
		}
		m.isWebsocket = true
	} else {
		if req.Kind() != reflect.Ptr || req.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("the type of second argment in %s should be pointer to struct", m.path)
		}
		if rsp.Kind() != reflect.Ptr || rsp.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("the type of third argment in %s should be pointer to struct", m.path)
		}
	}

	ret := method.Out(0)
	if ret.PkgPath() != "" || ret.Name() != "error" {
		return fmt.Errorf("return type in %s should be error", m.path)
	}

	methodValue := reflect.ValueOf(m.method)
	callFunc := func(ctx context.Context, req, rsp interface{}) error {
		// args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req), reflect.ValueOf(rsp)}
		args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req), reflect.ValueOf(rsp)}
		retValues := methodValue.Call(args)
		ret := retValues[0].Interface()
		if ret != nil {
			// ingore close error message
			if _, ok := ret.(*websocket.CloseError); ok {
				return nil
			}
			return ret.(error)
		}
		return nil
	}

	m.factory = func() (middleware.MethodFunc, interface{}, interface{}) {
		var rspVal, reqVal interface{}
		if m.isWebsocket {
			reqVal = &streamImp{}
			rspVal = reqVal
		} else {
			reqVal = reflect.New(req.Elem()).Interface()
			rspVal = reflect.New(rsp.Elem()).Interface()
		}
		return callFunc, reqVal, rspVal
	}
	if m.isWebsocket {
		m.reqType = req
		m.rspType = rsp
	} else {
		m.reqType = req.Elem()
		m.rspType = rsp.Elem()
	}
	return nil
}

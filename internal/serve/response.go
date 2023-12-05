package serve

import (
	"fmt"
	"net/url"

	json "github.com/goccy/go-json"
	"github.com/gorilla/schema"
	"github.com/ottstack/gofunc/pkg/ecode"
	"github.com/valyala/fasthttp"
)

var encoder = json.Marshal
var jsonDecoder = json.Unmarshal
var queryDecoder = func(queryStr []byte, v interface{}) error {
	u, err := url.ParseQuery(string(queryStr))
	if err != nil {
		return err
	}
	d := schema.NewDecoder()
	d.SetAliasTag(filedNameTag)
	d.IgnoreUnknownKeys(true)
	return d.Decode(v, u)
}

func writeErrResponse(w *fasthttp.RequestCtx, err error) {
	if _, ok := err.(*ecode.APIError); !ok {
		err = ecode.Errorf(500, err.Error())
	}
	w.Response.SetStatusCode(ecode.ToHttpCode(err))
	bs, _ := encoder(err)
	fmt.Println("")
	w.Write(bs)
}

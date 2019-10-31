package controller

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"io"
	"net/http"
)

type RequestForwarder struct {
	app     *iris.Application
	context context.Context
	request *http.Request
	response *http.Response
}

/* 转发用户context的RequestHeader到目标http请求 */
func (self *RequestForwarder) HandleRequestHeader() {
	for k, v := range self.context.Request().Header {
		self.request.Header.Set(k, v[0])
		self.app.Logger().Debug("Request Header: ", k, " -> ", v[0])
	}
}

/* 转发单个自定义RequestHeader到目标http请求 */
func(self *RequestForwarder) SetCustomRequestHeader(key, value string) {
	self.request.Header.Set(key, value)
	self.app.Logger().Debug("Custom Request Header: ", key, " -> ", value)
}

/* 转发多个自定义RequestHeader到目标http请求 */
func(self *RequestForwarder) SetCustomRequestHeaderMap(headers []map[string]interface{}) {
	for i:=0; i<len(headers); i++ {
		for k, v := range headers[i] {
			self.request.Header.Set(k, v.(string))
			self.app.Logger().Debug("Custom Request Header: ", k, " -> ", v)
		}
	}
}

/* 建立于目标URL的HTTP/HTTPS连接 */
func(self *RequestForwarder) Do() (err error) {
	resp, err := http.DefaultClient.Do(self.request)
	if err != nil {
		return err
	}
	self.response = resp
	return nil
}

/* 转发目标网站发来的http response header字段到用户context */
func (self *RequestForwarder) HandleResponseHeader() {
	for k, v := range self.response.Header {
		if k != "Server" {
			self.context.Header(k, v[0])
		}
		self.app.Logger().Debug("Response Header: ", k, " -> ", v[0])
	}
}

func(self *RequestForwarder) GetStatusCode() int {
	return self.response.StatusCode
}

func(self *RequestForwarder) GetBody() io.ReadCloser {
	return self.response.Body
}

func NewRequestForwarder(app *iris.Application, context context.Context, url string) (*RequestForwarder, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return &RequestForwarder{
		app: app,
		context: context,
		request: req,
	}, nil
}

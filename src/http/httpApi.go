package http

import (
	"github.com/buaazp/fasthttprouter"
	"github.com/json-iterator/go"
	"github.com/logs"
	"github.com/valyala/fasthttp"
	"global"
	"units"
)

type ApiHttp struct {
	router *fasthttprouter.Router
	logObj *logs.BeeLogger
	apiMap map[string] fasthttp.RequestHandler
}
//public

//end public

//private
func (obj *ApiHttp) run(addr string) {
	obj.logObj.Debug("http server will listen:%s", addr)
	if err := fasthttp.ListenAndServe(addr, obj.router.Handler); err != nil {
		obj.logObj.Warning("http server [%s] listen fail:%s", addr, err.Error())
		return
	} else {
		obj.logObj.Warning("http server [%s] listen success", addr)
	}
}
func (obj *ApiHttp) initMember(logName string) {
	obj.apiMap = make(map[string] fasthttp.RequestHandler)
	obj.router = fasthttprouter.New()
	obj.logObj = units.CreateLogobject(logName, global.GConfig.LogInfo.LogLevel, global.GConfig.LogInfo.LogMode)
}
func (obj *ApiHttp) initRouter() {
	for key, value := range obj.apiMap {
		if len(key) > 0 {
			obj.router.GET(key, value)
			obj.router.POST(key, value)
			obj.router.OPTIONS(key, obj.handleOptions)
		}
	}
}
func (obj *ApiHttp) handleOptions(ctx *fasthttp.RequestCtx){
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*") /* 处理跨域 */
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
}
func (obj *ApiHttp) useBaseResponse(ctx *fasthttp.RequestCtx, statusCode int, status, contentType string, httpCode int) {
	obj.handleOptions(ctx)
	var tmp CommResponse
	tmp.Status = status
	tmp.StatusCode = statusCode
	data,_ := jsoniter.Marshal(&tmp)
	ctx.SetBody(data)
	ctx.SetStatusCode(httpCode)
	ctx.SetContentType(contentType)
}
func (obj *ApiHttp) responseData(ctx *fasthttp.RequestCtx, data []byte, contentType string, httpCode int)  {
	obj.handleOptions(ctx)
	ctx.SetBody(data)
	ctx.SetStatusCode(httpCode)
	ctx.SetContentType(contentType)
}
//end private
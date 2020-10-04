package http

import (
	"fmt"
	"github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
	"global"
	"strings"
)

//public
type NetProxyHttpCmdSer struct {
	ApiHttp
	outChanObj chan global.HttpToProxyCmdSerP
}
func NewNetProxyHttpCmdSer(outChanObj chan global.HttpToProxyCmdSerP) *NetProxyHttpCmdSer {
	obj := &NetProxyHttpCmdSer{outChanObj:outChanObj}
	return obj
}
func (obj* NetProxyHttpCmdSer) Run() {
	obj.initMember("httpCmdSer")
	obj.initApiMap()
	obj.initRouter()

	httpPort := strings.Split(global.GConfig.CmdHttpSerAddr, ":")[1]
	obj.run("0.0.0.0:" + httpPort)
}
func (obj *NetProxyHttpCmdSer) Stop() {

}
//end public

//private
func (obj *NetProxyHttpCmdSer) initApiMap() {
	obj.apiMap[HAddUser] = obj.handleUserAdd
	obj.apiMap[HDelUser] = obj.handleUserDel
	obj.apiMap[HUpdateUser] = obj.handleUserUpdate
	obj.apiMap[HQueryUserList] = obj.handleGetAllUsers
	obj.apiMap[HSetTransmit] = obj.handleTransmitSet
	obj.apiMap[HDelTransmit] = obj.handleTransmitDel
	obj.apiMap[HGetTransmit] = obj.handleTransmitGet
	obj.apiMap[HUpdateTransmit] = obj.handleTransmitUpdate
}
func (obj *NetProxyHttpCmdSer) handleUserAdd(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "POST" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not POST", string(ctx.Method()))
		return
	}
	obj.logObj.Debug("handleUserAdd body:%s", string(ctx.PostBody()))
	var jsonObj global.UserInfo
	err_ := jsoniter.Unmarshal(ctx.PostBody(), &jsonObj)
	if nil != err_ {
		obj.logObj.Warning("[%s] Unmarshal fail:%s", string(ctx.PostBody()), err_.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("[%s] Unmarshal fail:%s", string(ctx.PostBody()),
			err_.Error()), ContentTypeJson, fasthttp.StatusOK)
		return
	}
	err := global.AddUser(jsonObj)
	if nil != err {
		obj.logObj.Warning("AddUser fail:%s", err.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("AddUser fail:%s", err.Error()), ContentTypeJson, fasthttp.StatusOK)
	} else {
		obj.logObj.Debug("AddUser %s success", jsonObj.UserName)
		obj.useBaseResponse(ctx, 1, fmt.Sprintf("AddUser %s success", jsonObj.UserName), ContentTypeJson, fasthttp.StatusOK)
	}
}
func (obj *NetProxyHttpCmdSer) handleUserDel(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "GET" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not GET", string(ctx.Method()))
		return
	}
	userName := ctx.QueryArgs().Peek("userName")
	if nil == userName {
		obj.logObj.Warning("parm userName is nil")
		obj.useBaseResponse(ctx, 0, "parm userName is nil", ContentTypeJson, fasthttp.StatusOK)
		return
	}
	obj.logObj.Debug("handleUserDel userName:%s", string(userName))
	err := global.DelUser(string(userName))
	if nil != err {
		obj.logObj.Warning("DelUser fail:%s", err.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("DelUser fail:%s", err.Error()), ContentTypeJson, fasthttp.StatusOK)
	} else {
		msg := global.HttpToProxyCmdSerP{}
		msg.CmdType = global.DelProxyClientUser
		msg.Data = string(userName)
		obj.outChanObj <- msg

		obj.logObj.Debug("DelUser %s success", string(userName))
		obj.useBaseResponse(ctx, 1, fmt.Sprintf("DelUser %s success", string(userName)), ContentTypeJson, fasthttp.StatusOK)
	}
}
func (obj *NetProxyHttpCmdSer) handleUserUpdate(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "POST" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not POST", string(ctx.Method()))
		return
	}
	obj.logObj.Debug("handleUserUpdate body:%s", string(ctx.PostBody()))
	var jsonObj global.UserInfo
	err_ := jsoniter.Unmarshal(ctx.PostBody(), &jsonObj)
	if nil != err_ {
		obj.logObj.Warning("[%s] Unmarshal fail:%s", string(ctx.PostBody()), err_.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("[%s] Unmarshal fail:%s", string(ctx.PostBody()),
			err_.Error()), ContentTypeJson, fasthttp.StatusOK)
		return
	}

	err := global.UpdateUser(jsonObj)
	if nil != err {
		obj.logObj.Warning("UpdateUser fail:%s", err.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("UpdateUser fail:%s", err.Error()), ContentTypeJson, fasthttp.StatusOK)
	} else {
		msg := global.HttpToProxyCmdSerP{}
		msg.CmdType = global.DelProxyClientUser
		msg.Data = jsonObj.UserName
		obj.outChanObj <- msg

		obj.logObj.Debug("UpdateUser %s success", jsonObj.UserName)
		obj.useBaseResponse(ctx, 1, fmt.Sprintf("UpdateUser %s success", jsonObj.UserName), ContentTypeJson, fasthttp.StatusOK)
	}
}
func (obj *NetProxyHttpCmdSer) handleGetAllUsers(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "GET" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not GET", string(ctx.Method()))
		return
	}
	obj.logObj.Debug("handleGetAllUsers")
	obj.responseData(ctx, getAllUserData(), ContentTypeJson, fasthttp.StatusOK)
}
func (obj *NetProxyHttpCmdSer) handleTransmitSet(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "POST" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not POST", string(ctx.Method()))
		return
	}

	obj.logObj.Debug("handleTransmitSet body:%s", string(ctx.PostBody()))

	var jsonObj TransmitSetP
	err_ := jsoniter.Unmarshal(ctx.PostBody(), &jsonObj)
	if nil != err_ {
		obj.logObj.Warning("[%s] Unmarshal fail:%s", string(ctx.PostBody()), err_.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("[%s] Unmarshal fail:%s", string(ctx.PostBody()),
			err_.Error()), ContentTypeJson, fasthttp.StatusOK)
		return
	}

	var retJsonObj TransmitSetResp
	retJsonObj.Result.StatusCode = 1
	retJsonObj.Result.Status = "OK"

	if !global.FindUser(jsonObj.UserName) {
		obj.logObj.Warning("user %s not find", jsonObj.UserName)
		retJsonObj.Result.StatusCode = 0
		retJsonObj.Result.Status = fmt.Sprintf("user %s not find", jsonObj.UserName)
		data,_ := jsoniter.Marshal(&retJsonObj)
		obj.responseData(ctx, data, ContentTypeJson, fasthttp.StatusOK)
		return
	}

	outAddr, inAddr := global.GetPortPair()
	if "" == outAddr || "" == inAddr {
		obj.logObj.Warning("GetPortPair fail")
		retJsonObj.Result.StatusCode = 0
		retJsonObj.Result.Status = "GetPortPair fail"
		data,_ := jsoniter.Marshal(&retJsonObj)
		obj.responseData(ctx, data, ContentTypeJson, fasthttp.StatusOK)
		return
	}

	oneSerPair := global.SerPair{PSerInnerAddr:inAddr, PSerOuterAddr:outAddr, UserName:jsonObj.UserName,
		Desc:jsonObj.Desc, TerAddr:jsonObj.TerAddr}
	err := global.AddSerPair(oneSerPair)
	if nil != err {
		retJsonObj.Result.StatusCode = 0
		retJsonObj.Result.Status = fmt.Sprintf("AddSerPair fail:%s", err.Error())
		obj.logObj.Warning("%s", retJsonObj.Result.Status)
		data,_ := jsoniter.Marshal(&retJsonObj)
		obj.responseData(ctx, data, ContentTypeJson, fasthttp.StatusOK)
		return
	} else {
		obj.logObj.Debug("AddSerPair [Out:%s,In:%s,Ter:%s] success", oneSerPair.PSerOuterAddr, oneSerPair.PSerInnerAddr, oneSerPair.TerAddr)
	}

	msg := global.HttpToProxyCmdSerP{}
	msg.CmdType = global.MAddSerPair
	msg.Data = oneSerPair
	obj.outChanObj <- msg

	retJsonObj.TInfo.PSerOuterAddr = oneSerPair.PSerOuterAddr
	retJsonObj.TInfo.TerAddr = oneSerPair.TerAddr
	retJsonObj.TInfo.Desc = oneSerPair.Desc
	data,_ := jsoniter.Marshal(&retJsonObj)
	obj.responseData(ctx, data, ContentTypeJson, fasthttp.StatusOK)
}
func (obj *NetProxyHttpCmdSer) handleTransmitDel(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "POST" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not POST", string(ctx.Method()))
		return
	}

	obj.logObj.Debug("handleTransmitDel body:%s", string(ctx.PostBody()))

	var jsonObj TransmitInfo
	err_ := jsoniter.Unmarshal(ctx.PostBody(), &jsonObj)
	if nil != err_ {
		obj.logObj.Warning("[%s] Unmarshal fail:%s", string(ctx.PostBody()), err_.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("[%s] Unmarshal fail:%s", string(ctx.PostBody()),
			err_.Error()), ContentTypeJson, fasthttp.StatusOK)
		return
	}

	oneSerPair, err := global.DelSerPair(jsonObj.PSerOuterAddr)
	if nil != err {
		obj.logObj.Warning("DelSerPair [%s] fail:%s", jsonObj.PSerOuterAddr, err.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("DelSerPair [%s] fail:%s", jsonObj.PSerOuterAddr, err.Error()),
			ContentTypeJson, fasthttp.StatusOK)
	} else {
		msg := global.HttpToProxyCmdSerP{}
		msg.CmdType = global.MDelSerPair
		msg.Data = oneSerPair
		obj.outChanObj <- msg

		obj.logObj.Debug("DelSerPair [%s] success", jsonObj.PSerOuterAddr)
		obj.useBaseResponse(ctx, 1, fmt.Sprintf("DelSerPair [%s] success", jsonObj.PSerOuterAddr),
			ContentTypeJson, fasthttp.StatusOK)
	}
}
func (obj *NetProxyHttpCmdSer) handleTransmitGet(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "GET" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not GET", string(ctx.Method()))
		return
	}
	userName := ctx.QueryArgs().Peek("userName")
	if nil == userName {
		obj.logObj.Warning("parm userName is nil")
		obj.useBaseResponse(ctx, 0, "parm userName is nil", ContentTypeJson, fasthttp.StatusOK)
		return
	}

	obj.logObj.Debug("handleTransmitGet userName:%s", string(userName))

	obj.responseData(ctx, getUserSerPair(string(userName)), ContentTypeJson, fasthttp.StatusOK)
}
func (obj *NetProxyHttpCmdSer) handleTransmitUpdate(ctx *fasthttp.RequestCtx) {
	if !obj.checkAuth(ctx) {
		obj.logObj.Warning("checkAuth fail")
		return
	}
	if "POST" != string(ctx.Method()) {
		obj.logObj.Warning("[%s] not POST", string(ctx.Method()))
		return
	}

	obj.logObj.Debug("handleTransmitUpdate body:%s", string(ctx.PostBody()))

	var jsonObj TransmitInfo
	err_ := jsoniter.Unmarshal(ctx.PostBody(), &jsonObj)
	if nil != err_ {
		obj.logObj.Warning("[%s] Unmarshal fail:%s", string(ctx.PostBody()), err_.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("[%s] Unmarshal fail:%s", string(ctx.PostBody()),
			err_.Error()), ContentTypeJson, fasthttp.StatusOK)
		return
	}

	oneSerPair, err := global.UpdateSerPair(jsonObj.PSerOuterAddr, jsonObj.Desc, jsonObj.TerAddr)
	if nil != err {
		obj.logObj.Warning("UpdateSerPair [%s] fail:%s", jsonObj.PSerOuterAddr, err.Error())
		obj.useBaseResponse(ctx, 0, fmt.Sprintf("UpdateSerPair [%s] fail:%s", jsonObj.PSerOuterAddr, err.Error()),
			ContentTypeJson, fasthttp.StatusOK)
	} else {
		msg1 := global.HttpToProxyCmdSerP{}
		msg1.CmdType = global.MDelSerPair
		msg1.Data = oneSerPair
		obj.outChanObj <- msg1

		msg2 := global.HttpToProxyCmdSerP{}
		msg2.CmdType = global.MAddSerPair
		msg2.Data = oneSerPair
		obj.outChanObj <- msg2

		obj.logObj.Debug("UpdateSerPair [%s] success", jsonObj.PSerOuterAddr)
		obj.useBaseResponse(ctx, 1, fmt.Sprintf("UpdateSerPair [%s] success", jsonObj.PSerOuterAddr),
			ContentTypeJson, fasthttp.StatusOK)
	}
}
func (obj *NetProxyHttpCmdSer) checkAuth(ctx *fasthttp.RequestCtx) bool {/* true=ok,false=fail */
	str_ := ctx.Request.Header.Peek(Token)
	if nil == str_ {
		obj.useBaseResponse(ctx, 0, "the header not contain Token", ContentTypeJson, fasthttp.StatusOK)
		return false
	} else {
		if string(str_) == global.GConfig.Token {
			return true
		} else {
			obj.useBaseResponse(ctx, 0, "Token error", ContentTypeJson, fasthttp.StatusUnauthorized)
			return false
		}
	}
}
//end private

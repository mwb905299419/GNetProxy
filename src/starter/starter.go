package starter

import (
	"global"
	"http"
	"proxySer"
)
//public
type Starter struct {
	httpCmdSObj *http.NetProxyHttpCmdSer
	tcpCmdSObj *proxySer.ProxySerCmd
	chanObj chan global.HttpToProxyCmdSerP
}
func NewStarter() *Starter {
	global.InitGlobalConfig()
	obj := &Starter{}
	obj.chanObj = make(chan global.HttpToProxyCmdSerP, global.MsgChanSize)
	obj.tcpCmdSObj = proxySer.NewProxySerCmd(obj.chanObj)
	obj.httpCmdSObj = http.NewNetProxyHttpCmdSer(obj.chanObj)
	return obj
}
func (obj *Starter) Run() {
	obj.tcpCmdSObj.Run()
	obj.httpCmdSObj.Run()
}
func (obj *Starter) Stop() {
	obj.tcpCmdSObj.Stop()
	obj.httpCmdSObj.Stop()
}
//end public

//private

//end private

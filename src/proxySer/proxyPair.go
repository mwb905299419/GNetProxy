package proxySer

import (
	"github.com/logs"
	"global"
	"units"
)

type ProxyPair struct {
	serOuterObj *PSerOuter
	serInnerObj *PSerInner
	logObj *logs.BeeLogger
	msgChan1, msgChan2 chan global.ChanDataParm
	outSerAddr, innerSerAddr string
}
func NewProxyPair(outSerAddr, innerSerAddr string, logMode, logLevel int) *ProxyPair {
	obj := &ProxyPair{}
	obj.logObj = units.CreateLogobject("proxySer", logLevel, logMode)
	obj.msgChan1 = make(chan global.ChanDataParm, global.MsgChanSize)
	obj.msgChan2 = make(chan global.ChanDataParm, global.MsgChanSize)
	obj.serInnerObj = nil
	obj.serOuterObj = nil
	obj.outSerAddr = outSerAddr
	obj.innerSerAddr = innerSerAddr
	return obj
}
func (obj *ProxyPair) Run() {
	obj.serInnerObj = NewPSerInner(obj.innerSerAddr, obj.logObj, obj.msgChan1, obj.msgChan2)
	obj.serOuterObj = NewPSerOuter(obj.outSerAddr, obj.logObj, obj.msgChan2, obj.msgChan1)
	obj.serInnerObj.Run()
	obj.serOuterObj.Run()
}
func (obj *ProxyPair) Stop() {
	if nil != obj.serInnerObj {
		obj.serInnerObj.Stop()
		obj.serInnerObj = nil
	}
	if nil != obj.serOuterObj {
		obj.serOuterObj.Stop()
		obj.serOuterObj = nil
	}
}

package proxyClient

import (
	"crypto/md5"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/logs"
	"global"
	"os"
	"units"
)

type ProxyClientCmd struct {
	logObj *logs.BeeLogger
	pcList global.ProxyClientP
	lcCmdClient *LcClientParm
	lcCmdChan chan global.ChanDataParm
	exit chan bool
	sendBuf []byte
	proxyClientMapObj map[string] *ProxyClient //key is PSerInnerAddr
	initPcList bool
}
func NewProxyClientCmd() *ProxyClientCmd {
	if err := getConfigInfo(); nil != err {
		panic(err)
	}
	obj := &ProxyClientCmd{}

	getProxyClientList(&obj.pcList)

	obj.logObj = units.CreateLogobject("ProxyClientCmd", GConfig.LogLevel, GConfig.LogMode)
	obj.lcCmdChan =  make(chan global.ChanDataParm, global.MsgChanSize)
	obj.lcCmdClient = NewTcpClient(GConfig.UserName, GConfig.CmdSerAddr, obj.logObj, obj.lcCmdChan, obj.lcNetState)
	obj.exit = make(chan bool, 1)
	obj.sendBuf = make([]byte, global.MaxBufSize + 33)
	obj.proxyClientMapObj = make(map[string] *ProxyClient)
	obj.initPcList = false

	return obj
}
func (obj *ProxyClientCmd) Run() {
	go obj.listenCmdMsg()
	obj.lcCmdClient.Run()
}
func (obj *ProxyClientCmd) Stop() {
	if nil != obj.lcCmdClient {
		obj.lcCmdClient.Stop()
		obj.lcCmdClient = nil
	}
	obj.exit <- true
}
func (obj *ProxyClientCmd) lcNetState(state int) {
	if 1 == state {
		var loginData global.ProxyClientLoginReqData
		loginData.UserName = GConfig.UserName

		data := []byte(GConfig.UserName + ":" + GConfig.UserPwd)
		has := md5.Sum(data)
		loginData.MD5Str = fmt.Sprintf("%x", has) //将[]byte转成16进制
		udata, _ := jsoniter.Marshal(&loginData)
		sendLen := global.OrgChanDataMsg(global.ProxyClientLoginReq, obj.lcCmdClient.clientId, udata, len(udata), obj.sendBuf)
		obj.lcCmdClient.sendTTData(obj.sendBuf, sendLen)
	}
}
func (obj *ProxyClientCmd) listenCmdMsg() {
	for {
		select {
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		case msg := <- obj.lcCmdChan:
			obj.handleMsg(&msg)
		}
	}
}
func (obj *ProxyClientCmd) handleMsg(msg *global.ChanDataParm) {
	if nil == msg {
		return
	}
	if global.ProxyClientLoginRep == msg.CmdType {
		var rep global.ProxyClientLoginRepData
		err := jsoniter.Unmarshal(msg.Data[:msg.DataLen], &rep)
		if nil != err {
			obj.logObj.Warning("unmarshal [%s] fail:%s", string(msg.Data[:msg.DataLen]), err.Error())
		} else {
			if 0 == rep.Result {
				obj.logObj.Warning("[%s] login fail", GConfig.UserName)
				os.Exit(0)
			} else {
				obj.logObj.Debug("[%s] login success", GConfig.UserName)
				if false == obj.initPcList {
					obj.initPcList = true
					for _, oneMap := range obj.pcList.TerAddrMapList {
						obj.addProxyClient(oneMap)
					}
				}
			}
		}
	} else if global.QueryProxyClientInfo == msg.CmdType {
		udata, _ := jsoniter.Marshal(&obj.pcList)
		dataLen := global.OrgChanDataMsg(global.RepProxyClientInfo, obj.lcCmdClient.clientId, udata, len(udata), obj.sendBuf)
		obj.lcCmdClient.sendTTData(obj.sendBuf, dataLen)
	} else if global.AddProxyClientInfo == msg.CmdType {
		obj.addProxyClientPair(msg)
	} else if global.DelProxyClientInfo == msg.CmdType {
		obj.delProxyClientPair(msg)
	} else if global.SyncProxyClientInfo == msg.CmdType || global.SyncProxyClientExit == msg.CmdType {
		var syncMap global.ProxyClientP
		err := jsoniter.Unmarshal(msg.Data[:msg.DataLen], &syncMap)
		if nil != err {
			obj.logObj.Warning("Unmarshal [%s] fail:%s", string(msg.Data[:msg.DataLen]), err.Error())
		} else {
			obj.handleSyncProxyClientInfo(&syncMap)
			if global.SyncProxyClientExit == msg.CmdType {
				obj.logObj.Debug("recv exit cmd")
				os.Exit(0)
			}
		}
	}
}
func (obj *ProxyClientCmd) handleSyncProxyClientInfo(syncMap *global.ProxyClientP) {
	if nil == syncMap {
		return
	}
	var alreadyExists bool
	for _,oneMap := range syncMap.TerAddrMapList {
		alreadyExists = false
		for _, oldOneMap := range obj.pcList.TerAddrMapList {
			if oneMap.PSerInnerAddr == oldOneMap.PSerInnerAddr {
				alreadyExists = true
				if oneMap.TerAddr != oldOneMap.TerAddr {//更新
					obj.doDelProxyClientPair(oldOneMap)
					obj.doAddProxyClientPair(oneMap)
				}
				break
			}
		}
		if !alreadyExists {
			obj.pcList.TerAddrMapList = append(obj.pcList.TerAddrMapList, oneMap)
			obj.addProxyClient(oneMap)
		}
	}
	for index_ := 0; index_ < len(obj.pcList.TerAddrMapList); {
		alreadyExists = false
		for _, oneMap := range syncMap.TerAddrMapList {
			if obj.pcList.TerAddrMapList[index_].PSerInnerAddr == oneMap.PSerInnerAddr {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			pclientObj,ok := obj.proxyClientMapObj[obj.pcList.TerAddrMapList[index_].PSerInnerAddr]
			if ok {
				obj.logObj.Debug("del PserInnerAddr:%s success", obj.pcList.TerAddrMapList[index_].PSerInnerAddr)
				pclientObj.Stop()
				delete(obj.proxyClientMapObj, obj.pcList.TerAddrMapList[index_].PSerInnerAddr)
				obj.pcList.TerAddrMapList = append(obj.pcList.TerAddrMapList[:index_], obj.pcList.TerAddrMapList[index_+1:]...)
			}
		} else {
			index_++
		}
	}
	saveProxyClientMap(&obj.pcList)
}
func (obj *ProxyClientCmd) addProxyClient(oneMap global.ProxyClientMapP) {
	_,ok := obj.proxyClientMapObj[oneMap.PSerInnerAddr]
	if ok {
		obj.logObj.Warning("addProxyClient %s already exists", oneMap.PSerInnerAddr)
		return
	}
	pclientObj := NewProxyClient(oneMap.PSerInnerAddr, oneMap.TerAddr, GConfig.LogMode, GConfig.LogLevel)
	if nil != pclientObj {
		obj.logObj.Debug("add PserInnerAddr:%s, terAddr:%s, Desc:%s success", oneMap.PSerInnerAddr, oneMap.TerAddr, oneMap.Desc)
		obj.proxyClientMapObj[oneMap.PSerInnerAddr] = pclientObj
		pclientObj.Run()
	} else {
		obj.logObj.Warning("NewProxyClient %s fail", oneMap.PSerInnerAddr)
	}
}
func (obj *ProxyClientCmd) addProxyClientPair(msg *global.ChanDataParm) {
	var oneMap global.ProxyClientMapP
	err := jsoniter.Unmarshal(msg.Data[:msg.DataLen], &oneMap)
	if nil != err {
		obj.logObj.Warning("Unmarshal [%s] fail:%s", string(msg.Data[:msg.DataLen]), err.Error())
	} else {
		obj.doAddProxyClientPair(oneMap)
	}
}
func (obj *ProxyClientCmd) doAddProxyClientPair(oneMap global.ProxyClientMapP) {
	err := putProxyClientMap(oneMap, &obj.pcList)
	if nil != err {
		obj.logObj.Warning("putProxyClientMap fail:%s", err.Error())
	} else {
		obj.addProxyClient(oneMap)
	}
}
func (obj *ProxyClientCmd) delProxyClientPair(msg *global.ChanDataParm) {
	var oneMap global.ProxyClientMapP
	err := jsoniter.Unmarshal(msg.Data[:msg.DataLen], &oneMap)
	if nil != err {
		obj.logObj.Warning("Unmarshal [%s] fail:%s", string(msg.Data[:msg.DataLen]), err.Error())
	} else {
		obj.doDelProxyClientPair(oneMap)
	}
}
func (obj *ProxyClientCmd) doDelProxyClientPair(oneMap global.ProxyClientMapP) {
	pclientObj,ok := obj.proxyClientMapObj[oneMap.PSerInnerAddr]
	if ok {
		obj.logObj.Debug("del PserInnerAddr:%s success", oneMap.PSerInnerAddr)
		pclientObj.Stop()
		delete(obj.proxyClientMapObj, oneMap.PSerInnerAddr)
		delProxyClientMap(oneMap, &obj.pcList)
	}
}
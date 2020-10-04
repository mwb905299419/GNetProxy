package proxySer

import (
	"github.com/json-iterator/go"
	"github.com/logs"
	"global"
	"strings"
	"tcp"
	"time"
	"units"
)
//public
type ProxySerCmd struct {
	inChanObj chan global.HttpToProxyCmdSerP
	serObj *tcp.TcpSer
	logObj *logs.BeeLogger
	proxyPairSerMap map[string] *ProxyPair //key is PSerOuterAddr
	timeoutSendBuf, cmdSendBuf []byte
	exit chan bool
}
func NewProxySerCmd(inChanObj chan global.HttpToProxyCmdSerP) *ProxySerCmd {
	obj := &ProxySerCmd{inChanObj:inChanObj}
	obj.logObj = units.CreateLogobject("proxySerCmd", global.GConfig.LogInfo.LogLevel, global.GConfig.LogInfo.LogMode)
	cmdPort := strings.Split(global.GConfig.CmdSerAddr, ":")[1]
	obj.serObj = tcp.NewTcpSer(obj.logObj, "0.0.0.0:" + cmdPort, "ProxySerCmd", obj.clientData)
	obj.serObj.SetClientClosedCb(obj.ProxyCmdClientClosed)
	obj.timeoutSendBuf = make([]byte, global.MaxBufSize)
	obj.cmdSendBuf = make([]byte, global.MaxBufSize)
	obj.exit = make(chan bool, 1)
	obj.proxyPairSerMap = make(map[string] *ProxyPair)
	return obj
}
func (obj *ProxySerCmd) Run() {
	if nil == obj.serObj {
		return
	}
	go obj.listenChan()
	go obj.timeout()
	obj.runHistoryProxyPairSer()
	obj.serObj.Run()
}
func (obj *ProxySerCmd) Stop() {
	if nil != obj.serObj {
		obj.serObj.Stop()
		obj.serObj = nil
	}
	obj.exit <- true
}
//end public

//private
func (obj *ProxySerCmd) runHistoryProxyPairSer() {
	for _, oneSerPair := range global.GConfig.SerPairs.SerPairList {
		serPairObj := NewProxyPair(oneSerPair.PSerOuterAddr, oneSerPair.PSerInnerAddr, global.GConfig.LogInfo.LogMode,
			global.GConfig.LogInfo.LogLevel)
		serPairObj.Run()
		obj.proxyPairSerMap[oneSerPair.PSerOuterAddr] = serPairObj
		obj.logObj.Debug("[%s][%s] add success", oneSerPair.UserName, oneSerPair.PSerOuterAddr)
	}
}
func (obj *ProxySerCmd) timeout() {
	for {
		select {
		case <-time.After(20 * time.Second):
			obj.syncAllCmdClient()
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		}
	}
}
func (obj *ProxySerCmd) syncAllCmdClient() {
	obj.serObj.RwLock.Lock()
	defer obj.serObj.RwLock.Unlock()
	for _, client := range obj.serObj.ClientList {
		len := obj.orgSyncClientMsg(client.GetClientId(), global.SyncProxyClientInfo, obj.timeoutSendBuf)
		err := client.SendTTData(obj.timeoutSendBuf, len)
		if nil != err {
			obj.logObj.Warning("[%s][%s] send fail", client.GetClientId(), string(obj.timeoutSendBuf[:len]))
		} else {
			//obj.logObj.Debug("[%s][%s] send success", client.GetClientId(), string(data[:len]))
		}
	}
}
func (obj *ProxySerCmd) listenChan() {
	for {
		select {
		case msg := <- obj.inChanObj:
			obj.handleMsg(&msg)
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		}
	}
}
func (obj *ProxySerCmd) handleMsg(msg *global.HttpToProxyCmdSerP) {
	if nil == msg {
		return
	}
	if global.DelProxyClientUser == msg.CmdType {
		userName := msg.Data.(string)
		clientObj := obj.findCmdClient(userName)
		if nil != clientObj {
			len := obj.orgSyncClientMsg(userName, global.SyncProxyClientExit, obj.cmdSendBuf)
			err := clientObj.SendTTData(obj.cmdSendBuf, len)
			if nil != err {
				obj.logObj.Warning("[%s][%s] send fail", clientObj.GetClientId(), string(obj.cmdSendBuf[:len]))
			} else {
				obj.logObj.Debug("[%s][%s] send success", clientObj.GetClientId(), string(obj.cmdSendBuf[:len]))
			}
		} else {
			obj.logObj.Warning("[%s] offline", userName)
		}
	} else if global.MAddSerPair == msg.CmdType || global.MDelSerPair == msg.CmdType {
		oneSerPair := msg.Data.(global.SerPair)
		clientObj := obj.findCmdClient(oneSerPair.UserName)
		if nil != clientObj {
			var cmdType byte = global.AddProxyClientInfo
			if global.MDelSerPair == msg.CmdType {
				cmdType = global.DelProxyClientInfo
				serPairObj,ok := obj.proxyPairSerMap[oneSerPair.PSerOuterAddr]
				if ok {
					serPairObj.Stop()
					delete(obj.proxyPairSerMap, oneSerPair.PSerOuterAddr)
				} else {
					obj.logObj.Warning("[%s][%s] not exists", clientObj.GetClientId(), oneSerPair.PSerOuterAddr)
				}
			} else {
				serPairObj,ok := obj.proxyPairSerMap[oneSerPair.PSerOuterAddr]
				if ok {
					obj.logObj.Warning("[%s][%s] already exists", clientObj.GetClientId(), oneSerPair.PSerOuterAddr)
				} else {
					serPairObj = NewProxyPair(oneSerPair.PSerOuterAddr, oneSerPair.PSerInnerAddr, global.GConfig.LogInfo.LogMode,
						global.GConfig.LogInfo.LogLevel)
					serPairObj.Run()
					obj.proxyPairSerMap[oneSerPair.PSerOuterAddr] = serPairObj
					obj.logObj.Debug("[%s][%s] add success", clientObj.GetClientId(), oneSerPair.PSerOuterAddr)
				}
			}
			len := obj.orgProxyClientInfoMsg(cmdType, &oneSerPair, obj.cmdSendBuf)
			err := clientObj.SendTTData(obj.cmdSendBuf, len)
			if nil != err {
				obj.logObj.Warning("[%s][%s] send fail", clientObj.GetClientId(), string(obj.cmdSendBuf[:len]))
			} else {
				obj.logObj.Debug("[%s][%s] send success", clientObj.GetClientId(), string(obj.cmdSendBuf[:len]))
			}
		} else {
			obj.logObj.Warning("[%s] offline", oneSerPair.UserName)
		}
	}
}
func (obj *ProxySerCmd) clientData(data []byte, dataLen int) {
	msg := global.ParseChanDataMsg(data, dataLen)
	if global.ProxyClientLoginReq == msg.CmdType {
		var loginJson global.ProxyClientLoginReqData
		err := jsoniter.Unmarshal(msg.Data[0:msg.DataLen], &loginJson)

		var jsonObj global.ProxyClientLoginRepData
		jsonObj.Result = 0

		if nil != err {
			obj.logObj.Warning("Unmarshal [%s] fail:%s", string(msg.Data[0:msg.DataLen]), err.Error())
		} else {
			if global.UserAuth(loginJson.UserName, loginJson.MD5Str) {
				jsonObj.Result = 1
				obj.logObj.Debug("[%s] login success", msg.ClientId)
			} else {
				obj.logObj.Warning("[%s] login fail", msg.ClientId)
			}
		}

		data,_ := jsoniter.Marshal(&jsonObj)
		tmpBuf := make([]byte, 1024 * 4)
		len := global.OrgChanDataMsg(global.ProxyClientLoginRep, msg.ClientId, data, len(data), tmpBuf)
		client := obj.findCmdClient(msg.ClientId)
		if nil != client {
			err := client.SendTTData(tmpBuf, len)
			if nil != err {
				obj.logObj.Warning("[%s] send fail", msg.ClientId)
			}
		}
	}
}
func (obj *ProxySerCmd) orgSyncClientMsg(userName string, cmdType byte, outBuf []byte) int{
	global.GConfig.SerPairLock.Lock()
	defer global.GConfig.SerPairLock.Unlock()

	var jsonObj global.ProxyClientP
	for index_ := 0; index_ < len(global.GConfig.SerPairs.SerPairList); index_++ {
		if global.GConfig.SerPairs.SerPairList[index_].UserName == userName {
			oneClientTerMap := global.ProxyClientMapP{PSerInnerAddr:global.GConfig.SerPairs.SerPairList[index_].PSerInnerAddr,
				TerAddr:global.GConfig.SerPairs.SerPairList[index_].TerAddr, Desc:global.GConfig.SerPairs.SerPairList[index_].Desc}
			jsonObj.TerAddrMapList = append(jsonObj.TerAddrMapList, oneClientTerMap)
		}
	}
	data,_ := jsoniter.Marshal(&jsonObj)
	dataLen := global.OrgChanDataMsg(cmdType, userName, data, len(data), outBuf)
	return dataLen
}
func (obj *ProxySerCmd) orgProxyClientInfoMsg(cmdType byte, pairInfo *global.SerPair, outBuf []byte) int {
	var jsonObj global.ProxyClientMapP
	jsonObj.Desc = pairInfo.Desc
	jsonObj.TerAddr = pairInfo.TerAddr
	jsonObj.PSerInnerAddr = pairInfo.PSerInnerAddr
	data,_ := jsoniter.Marshal(&jsonObj)
	dataLen := global.OrgChanDataMsg(cmdType, pairInfo.UserName, data, len(data), outBuf)
	return dataLen
}
func (obj *ProxySerCmd) findCmdClient(clientId string) *tcp.TcpClient {
	obj.serObj.RwLock.Lock()
	defer obj.serObj.RwLock.Unlock()
	for _, client := range obj.serObj.ClientList {
		if client.GetClientId() == clientId {
			return client
		}
	}
	return nil
}
func (obj *ProxySerCmd) ProxyCmdClientClosed(clientId string) {
	global.UserOffline(clientId)
}
//end private
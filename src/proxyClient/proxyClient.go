package proxyClient

import (
	"encoding/base64"
	"github.com/logs"
	"global"
	"units"
)

type ProxyClient struct {
	lcClient *LcClientParm
	logObj *logs.BeeLogger
	lcClientChan, terClientChan chan global.ChanDataParm
	terClientMap map[string] *TerminalTcpClient
	terAddr string
	exit chan bool
	sendBuf []byte
}
func NewProxyClient(pserInnerAddr, terAddr string, logMode, logLevel int) *ProxyClient {
	obj := &ProxyClient{}
	obj.terAddr = terAddr
	obj.logObj = units.CreateLogobject("proxyClient", logLevel, logMode)
	obj.lcClientChan = make(chan global.ChanDataParm, global.MsgChanSize)
	obj.terClientChan = make(chan global.ChanDataParm, global.MsgChanSize)
	obj.lcClient = NewTcpClient(base64.StdEncoding.EncodeToString([]byte(pserInnerAddr)), pserInnerAddr, obj.logObj,
		obj.lcClientChan, nil)
	obj.terClientMap = make(map[string]*TerminalTcpClient)
	obj.exit = make(chan bool, 2)
	obj.sendBuf = make([]byte, global.MaxBufSize + 33)
	return obj
}
func (obj *ProxyClient) Run() {
	obj.lcClient.Run()
	go obj.lcClientData()
	go obj.terClientData()
}
func (obj *ProxyClient) Stop() {
	obj.lcClient.Stop()
	for key, terClient := range obj.terClientMap {
		terClient.Stop()
		delete(obj.terClientMap, key)
	}
	obj.exit <- true
	obj.exit <- true
}
//private
func (obj *ProxyClient) lcClientData() {
	for {
		select {
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		case msg := <- obj.lcClientChan:
			obj.handleLcClientMsg(&msg)
		}
	}
}
func (obj *ProxyClient) terClientData() {
	for {
		select {
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		case msg := <- obj.terClientChan:
			if global.Closed == msg.CmdType && msg.DataLen > 0 && 1 == msg.Data[0] {

			} else {
				obj.handleTerClientMsg(&msg)
			}
		}
	}
}
func (obj *ProxyClient) handleLcClientMsg(msg *global.ChanDataParm) {
	if global.Connected == msg.CmdType {
		terClient, ok := obj.terClientMap[msg.ClientId]
		if ok {
			terClient.Stop()
		}
		terClient = NewTerminalTcpClient(obj.logObj, obj.terAddr, msg.ClientId, obj.terClientChan)
		terClient.Run()
		obj.terClientMap[msg.ClientId] = terClient
	} else if global.Closed == msg.CmdType {
		terClient, ok := obj.terClientMap[msg.ClientId]
		if ok {
			terClient.Stop()
			delete(obj.terClientMap, msg.ClientId)
		}
	} else if global.Data == msg.CmdType {
		terClient, ok := obj.terClientMap[msg.ClientId]
		if ok {
			terClient.Send(msg.Data, msg.DataLen)
		}
	}
}
func (obj *ProxyClient) handleTerClientMsg(msg *global.ChanDataParm) {
	dataLen := global.OrgChanDataMsg(msg.CmdType, msg.ClientId, msg.Data, msg.DataLen, obj.sendBuf)
	obj.lcClient.sendTTData(obj.sendBuf, dataLen)
}
//end private

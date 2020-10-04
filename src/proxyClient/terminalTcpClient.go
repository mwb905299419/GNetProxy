package proxyClient

import (
	"github.com/logs"
	"global"
	"net"
)

type TerminalTcpClient struct {
	terminalAddr string
	logObj *logs.BeeLogger
	outChanMsg chan global.ChanDataParm
	terminalId string
	connObj net.Conn
	isAutoClose []byte
}
func NewTerminalTcpClient(logObj *logs.BeeLogger, terAddr, terminalId string, outChanMsg chan global.ChanDataParm) *TerminalTcpClient {
	obj := &TerminalTcpClient{terminalAddr:terAddr, terminalId:terminalId, logObj:logObj, outChanMsg:outChanMsg}
	obj.isAutoClose = make([]byte, 1)
	obj.isAutoClose[0] = 0
	return obj
}
func (obj *TerminalTcpClient) Run() {
	obj.doConnect()
}
func (obj *TerminalTcpClient) Stop() {
	obj.isAutoClose[0] = 1
	if nil != obj.connObj {
		obj.connObj.Close()
	}
}
func (obj *TerminalTcpClient) Send(data []byte, dataLen int) {
	if nil != obj.connObj && 1 != obj.isAutoClose[0] {
		slen, err_ := obj.connObj.Write(data[:dataLen])
		if nil != err_ {
			obj.logObj.Warning("send sub fail:%s", err_.Error())
		} else if dataLen != slen {
			obj.logObj.Warning("slen(%d) != len(%d)", slen, dataLen)
		} else {
			obj.logObj.Debug("TW:%d", slen)
		}
	} else {
		obj.logObj.Warning("[%s] connObj is nil", obj.terminalAddr)
	}
}
func (obj *TerminalTcpClient) doConnect() {
	var err_ error
	obj.connObj, err_ = net.Dial("tcp", obj.terminalAddr)
	if nil != err_ {
		obj.logObj.Warning("connect [%s] fail:%s", obj.terminalAddr, err_)
		obj.outputMsg(global.Closed, []byte(""), 0)
	} else {
		obj.logObj.Debug("connect [%s] success", obj.terminalAddr)
		go obj.doRead()
	}
}
func (obj_ *TerminalTcpClient) doRead() {
	recvBuf :=  make([]byte, global.InnerClientRecvSize)
	for {
		rsize_, err_ := obj_.connObj.Read(recvBuf[0:])
		if nil != err_ {
			obj_.logObj.Warning("[%s:%s] read data fail:%s", obj_.terminalId, obj_.terminalAddr, err_)
			break
		} else if rsize_ > 0 {
			obj_.outputMsg(global.Data, recvBuf, rsize_)
		} else {
			break
		}
	}
	obj_.outputMsg(global.Closed, obj_.isAutoClose, 1)
	obj_.logObj.Warning("[%s:%s] closed", obj_.terminalId, obj_.terminalAddr)
}
func (obj* TerminalTcpClient) outputMsg(cmdType byte, data[]byte, dataLen int) {
	msg := global.ChanDataParm{}
	msg.CmdType = cmdType
	msg.ClientId = obj.terminalId
	msg.DataLen = dataLen
	msg.Data = append(msg.Data, data[0:dataLen]...)
	obj.outChanMsg <- msg
	obj.logObj.Debug("TR:%d", dataLen)
}

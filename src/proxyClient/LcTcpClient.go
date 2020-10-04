package proxyClient

import (
	"github.com/logs"
	"global"
	"net"
	"time"
	"units"
)

/*
protocol stack format:
1          + 1         + 4                          + 1    + 1        + Data     + 1
FRAME_HEAD + FRAME_VER + 从第8字节到末尾减1的字节数 + crc8 + 消息类型 + 消息数据 + FRAME_TAIL
*/

const (
	CHECK_CONNECT_INTERVAL = 20
	SEND_HEART_INTERVAL = 20

	FRAME_HEAD = 0x7E
	FRAME_TAIL = 0x7F
	FRAME_VER = 0x01

	TT_CONNECT     = 0x10
	TT_CONNACK     = 0x20
	TT_PUBLISH     = 0x30
	TT_PUBACK      = 0x40
	TT_PUBREC      = 0x50
	TT_PUBREL      = 0x60
	TT_PUBCOMP     = 0x70
	TT_SUBSCRIBE   = 0x80
	TT_SUBACK      = 0x90
	TT_UNSUBSCRIBE = 0xA0
	TT_UNSUBACK    = 0xB0
	TT_PINGREQ     = 0xC0
	TT_PINGRESP    = 0xD0
	TT_DISCONNECT  = 0xE0
	TT_DATA        = 0xF0
	TT_EXECUTE     = 0xF1	/* 消息数据:可执行程序的名称(包括后缀名)+0+可执行程序运行需要的参数(JSON字符串) */
	TT_EXIT		   = 0xF2   /* 退出程序的通知 */

	TUPLE_MSG_SIG = 1258
	TUPLE_ERR = 0
	TUPLE_GET = 1
	TUPLE_PUT = 2
	TUPLE_APD = 3
	TUPLE_VAL = 4
	TUPLE_DEL = 5
	TUPLE_MAK = 6
	TUPLE_UMK = 7
)

type LcClientParm struct {
	clientId string
	cmdNum int
	serAddr string
	connObj net.Conn
	logObj *logs.BeeLogger
	curConnectState int /* 0=closed,1=connected */
	sendBuffer, tmpBuffer,recvCacheBuf, userData []byte
	recvCacheCount int
	checkConnectInterval, sendHeartInterval, heartTimeout int
	outChanMsg chan global.ChanDataParm
	exit chan bool
	netStateCb func(state int)
}
func NewTcpClient(clientId, serAddr string, logObj *logs.BeeLogger, outChanMsg chan global.ChanDataParm, netStateCb func(state int)) *LcClientParm{
	obj := &LcClientParm{}
	obj.exit = make(chan bool, 1)
	obj.serAddr = serAddr
	obj.logObj = logObj
	obj.outChanMsg = outChanMsg
	obj.cmdNum = 0
	obj.clientId = clientId
	obj.curConnectState = 0
	obj.recvCacheBuf = make([]byte, global.MaxBufSize)
	obj.recvCacheCount = 0
	obj.sendBuffer = make([]byte, global.MaxBufSize)
	obj.checkConnectInterval = CHECK_CONNECT_INTERVAL
	obj.sendHeartInterval = SEND_HEART_INTERVAL
	obj.heartTimeout = SEND_HEART_INTERVAL + 10
	obj.netStateCb = netStateCb
	return obj
}
func (obj_ *LcClientParm) Run(){
	go obj_.timeOut()
	obj_.doConnect()
}
func (obj *LcClientParm) Stop() {
	obj.curConnectState = -1
	obj.exit <- true
	if nil != obj.connObj {
		obj.connObj.Close()
	}
}
func (obj_ *LcClientParm) doRead() {
	if nil != obj_.connObj && 1 == obj_.curConnectState {
		for {
			rsize_, err_ := obj_.connObj.Read(obj_.recvCacheBuf[obj_.recvCacheCount:])
			if nil != err_ {
				obj_.logObj.Warning("[%s:%s] read data fail:%s", obj_.clientId, obj_.serAddr, err_)
				break
			} else if rsize_ > 0 {
				obj_.recvCacheCount = obj_.recvCacheCount + rsize_
				obj_.handleRead()
			} else {
				break
			}
		}
	}
	obj_.curConnectState = 0
	obj_.logObj.Warning("[%s:%s] closed", obj_.clientId, obj_.serAddr)
}
func (obj_ *LcClientParm) doConnect() {
	if -1 == obj_.curConnectState {
		return
	}
	var err_ error
	obj_.connObj, err_ = net.Dial("tcp", obj_.serAddr)
	if nil != err_ {
		obj_.curConnectState = 0
		obj_.logObj.Warning("connect [%s] fail:%s", obj_.serAddr, err_)
		if nil != obj_.netStateCb {
			obj_.netStateCb(0)
		}
	} else {
		obj_.curConnectState = 1
		obj_.logObj.Debug("connect [%s] success", obj_.serAddr)
		go obj_.doRead()
		obj_.sendHeart()
		if nil != obj_.netStateCb {
			obj_.netStateCb(1)
		}
	}
}
func (obj_ *LcClientParm) timeOut(){
	for {
		select {
		case <- time.After(1 * time.Second):
			obj_.checkConnect()
			obj_.sendKeepalive()
			obj_.checkHeartTimeout()
		case exitTag := <- obj_.exit:
			if exitTag {
				return
			}
		}
	}
}
func (obj_ *LcClientParm) checkHeartTimeout() {
	obj_.heartTimeout = obj_.heartTimeout - 1
	if obj_.heartTimeout < 1 {
		obj_.heartTimeout = SEND_HEART_INTERVAL + 10
		if nil != obj_.connObj && 1 == obj_.curConnectState {
			obj_.connObj.Close()
			obj_.curConnectState = 0
		}
	}
}
func (obj_ *LcClientParm) sendHeart() {
	obj_.orgProtoAndSend(TT_PINGREQ, []byte(obj_.clientId), len(obj_.clientId))
}
func (obj *LcClientParm) sendTTData(data []byte, dataLen int) {
	obj.orgProtoAndSend(TT_DATA, data, dataLen)
}
func (obj_ *LcClientParm) checkConnect() {
	obj_.checkConnectInterval = obj_.checkConnectInterval - 1
	if obj_.checkConnectInterval < 1 {
		obj_.checkConnectInterval = CHECK_CONNECT_INTERVAL
		if (0 == obj_.curConnectState){
			obj_.doConnect()
		}
	}
}
func (obj_ *LcClientParm) sendKeepalive(){
	obj_.sendHeartInterval = obj_.sendHeartInterval - 1
	if obj_.sendHeartInterval < 1 {
		obj_.sendHeartInterval = SEND_HEART_INTERVAL
		if (1 == obj_.curConnectState) {
			obj_.sendHeart()
		}
	}
}
func (obj_ *LcClientParm) handleRead(){
Check:
	if obj_.recvCacheCount < 8 {
		return
	}
	if FRAME_HEAD != obj_.recvCacheBuf[0] || FRAME_VER != obj_.recvCacheBuf[1] ||
		units.Crc8(obj_.recvCacheBuf, 6) != obj_.recvCacheBuf[6]{
		obj_.recvCacheCount = 0
		return
	}
	dataLen,_ := units.BytesToIntS(obj_.recvCacheBuf[2:2 + 4])
	if obj_.recvCacheCount < (dataLen + 8) {
		return
	}
	obj_.userData = obj_.recvCacheBuf[7:7 + dataLen]
	obj_.handleUserData(dataLen)
	copy(obj_.recvCacheBuf[0:], obj_.recvCacheBuf[8 + dataLen:obj_.recvCacheCount])
	obj_.recvCacheCount =obj_.recvCacheCount - (8 + dataLen)

	goto Check
}
func (obj_ *LcClientParm) handleUserData(dataLen int) {
	if dataLen < 1 {
		return
	}
	msgType := obj_.userData[0]
	switch msgType {
	case TT_EXECUTE:
		var execName string
		var index int = 1
		for ; index < dataLen; index++ {
			if 0x00 == obj_.userData[index] {
				break
			}
		}
		execName = string(obj_.userData[1:index])
		obj_.handleTTExecute(execName, string(obj_.userData[index + 1:dataLen - 1]))
	case TT_CONNACK:
	case TT_DATA:
		obj_.handleTTData(obj_.userData[1:dataLen - 1], dataLen - 2)
	case TT_PINGRESP:
		obj_.handlePingresq()
	}
}
func (obj *LcClientParm) handleTTData(data []byte, dataLen int) {
	if nil != obj.outChanMsg {
		obj.outChanMsg <- global.ParseChanDataMsg(data, dataLen)
	}
}
func (obj_ *LcClientParm) handleTTExecute(execName string, execParm string) {

}
func (obj_ *LcClientParm) handlePingresq() {
	obj_.heartTimeout = SEND_HEART_INTERVAL + 10
}
func (obj_ *LcClientParm) orgProtoAndSend(ptype byte, data_ []byte, dlen int) {
	_siz := 1 + dlen + 1
	obj_.sendBuffer[0] = FRAME_HEAD
	obj_.sendBuffer[1] = FRAME_VER
	obj_.sendBuffer[2] = (byte)(_siz)
	obj_.sendBuffer[3] = (byte)(_siz >> 8)
	obj_.sendBuffer[4] = (byte)(_siz >> 16)
	obj_.sendBuffer[5] = (byte)(_siz >> 24)
	obj_.sendBuffer[6] = units.Crc8(obj_.sendBuffer, 6)
	obj_.sendBuffer[7] = ptype
	obj_.sendBuffer = append(obj_.sendBuffer[:8], data_[:dlen]...)
	obj_.sendBuffer = append(obj_.sendBuffer[:_siz + 8 - 2], 0)
	obj_.sendBuffer = append(obj_.sendBuffer[:_siz + 8 - 1], FRAME_TAIL)

	obj_.send(obj_.sendBuffer, 1 + dlen + 1 + 8)
}
func (obj_ *LcClientParm) send(data []byte, len int) {
	if -1 == obj_.curConnectState {
		return
	}
	if  1 == obj_.curConnectState && nil != obj_.connObj {
		slen, err_ := obj_.connObj.Write(data[:len])
		if nil != err_ {
			obj_.logObj.Warning("send sub fail:%s", err_.Error())
		} else if len != slen {
			obj_.logObj.Warning("slen(%d) != len(%d)", slen, len)
		}
	}
}

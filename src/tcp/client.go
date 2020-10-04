package tcp

import (
	"fmt"
	"github.com/kataras/iris/core/errors"
	"github.com/logs"
	"net"
	"time"
	"units"
)

const (
	TcpClientTimeoutInterval = 30

	CHECK_CONNECT_INTERVAL = 20
	SEND_HEART_INTERVAL = 20

	MaxSendBufSize = (1024 * 1024 * 8)
	MaxRecvBufSize = (1024 * 1024 * 8)

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
	TT_EXIT        = 0xF2	/* 退出程序 */

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
type TcpClient struct {
	connObj net.Conn
	driveId string
	recvUpdateTime int64
	invalid bool
	log_ *logs.BeeLogger
	UseCount int
	sendBuffer, tmpBuffer,recvCacheBuf, userData []byte
	recvCacheCount int
	clientIdExists func(clientId string) bool
	dataCb func(data []byte, dataLen int)
}
/* 如果不需要 chan 传递消息，dataNotifyChan穿nil */
func newTcpClient(client net.Conn, log_ *logs.BeeLogger, clientIdExists func(clientId string) bool, dataCb func(data []byte, dataLen int)) *TcpClient{
	ret_ := &TcpClient{connObj:client, invalid:false, log_:log_, driveId:"",
		recvUpdateTime:time.Now().Unix(), UseCount:0, clientIdExists:clientIdExists, dataCb:dataCb}
	ret_.recvCacheBuf = make([]byte,MaxRecvBufSize)
	ret_.recvCacheCount = 0
	ret_.sendBuffer = make([]byte, MaxSendBufSize)
	ret_.tmpBuffer = make([]byte, 256)
	ret_.readData()
	return ret_
}
func (obj *TcpClient) closeTcpClient(){
	if nil != obj.connObj {
		obj.connObj.Close()
	}
}
func (obj *TcpClient) isTimeOut() bool {
	if time.Now().Unix() - obj.recvUpdateTime > TcpClientTimeoutInterval {
		return true
	} else {
		return false
	}
}
/*
发送消息运行指定程序
executeName:指定运行程序的名称
executeParm:用户自定义参数的JSON字符串，比如：里面可以包括会话ID，网络连接信息等
*/
func (obj *TcpClient) SendExecute(executeName, executeParm string) error {
	obj.tmpBuffer = append(obj.tmpBuffer[:0], []byte(executeName)...)
	obj.tmpBuffer = append(obj.tmpBuffer[:len(executeName)], 0)
	obj.tmpBuffer = append(obj.tmpBuffer[:len(executeName) + 1], []byte(executeParm)...)
	//executeName + 0 + executeParm
	return obj.orgProtoAndSend(TT_EXECUTE, obj.tmpBuffer, len(executeName) + 1 + len(executeParm))
}
/*
结束指定程序的运行
*/
func (obj* TcpClient) SendExit() error {
	return obj.orgProtoAndSend(TT_EXIT, []byte(obj.driveId), len(obj.driveId))
}
func (obj *TcpClient) GetClientId() string {
	return obj.driveId
}
/*
发送TT_DATA类型的数据
*/
func (obj *TcpClient) SendTTData(data []byte, len int) error {
	return obj.orgProtoAndSend(TT_DATA, data, len)
}
/************************************************ sprivate ************************************/
func (obj_ *TcpClient) readData(){
	if nil == obj_.connObj {
		return
	}
	go func() {
	LoopBreak:
		for {
			rsize_, err_ := obj_.connObj.Read(obj_.recvCacheBuf[obj_.recvCacheCount:])
			if nil != err_ {
				obj_.log_.Debug("[%s] recv closed,err:%s", obj_.connObj.RemoteAddr().String(), err_.Error())
				break LoopBreak
			} else if rsize_ > 0 {
				obj_.recvUpdateTime = time.Now().Unix()
				obj_.recvCacheCount = obj_.recvCacheCount + rsize_
				obj_.handleRead()
			} else {
				obj_.log_.Debug("[%s:%d] recv closed,cacheCount:%d", obj_.connObj.RemoteAddr().String(), rsize_, obj_.recvCacheCount)
				break LoopBreak
			}
		}
		obj_.invalid = true
	}()
}
func (obj_ *TcpClient) handleRead() {
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
func (obj_ *TcpClient) handleUserData(dataLen int) {
	if dataLen < 1 {
		return
	}
	msgType := obj_.userData[0]
	switch msgType {
	case TT_PINGREQ:
		obj_.handlePing(obj_.userData[1:dataLen - 1])
	case TT_DATA:
		obj_.handleTTData(obj_.userData[1:dataLen - 1], dataLen - 2)
	}
}
func (obj_ *TcpClient) handlePing(pingData []byte) {
	if 0 == len(obj_.driveId) {
		if nil != obj_.clientIdExists && true == obj_.clientIdExists(string(pingData)) {
			obj_.SendExit()
			return
		}
		obj_.driveId = string(pingData)
	}
	if err := obj_.orgProtoAndSend(TT_PINGRESP, pingData, len(pingData)); nil != err {
		obj_.log_.Warning("[%s] TT_PINGRESP fail:%s", obj_.driveId, err.Error())
	}
}
func (obj *TcpClient) handleTTData(data []byte, dataLen int) {
	if nil != obj.dataCb {
		obj.dataCb(data, dataLen)
	}
}
func (obj_ *TcpClient) orgProtoAndSend(ptype byte, data_ []byte, dlen int) error {
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

	return obj_.send(obj_.sendBuffer, 1 + dlen + 1 + 8)
}
func (obj_ *TcpClient) send(data []byte, len int) error {
	if  false == obj_.invalid && nil != obj_.connObj {
		slen, err_ := obj_.connObj.Write(data[:len])
		if nil != err_ {
			obj_.log_.Warning("send sub fail:%s", err_.Error())
			return err_
		} else if len != slen {
			str := fmt.Sprintf("slen(%d) != len(%d)", slen, len)
			obj_.log_.Warning("%s", str)
			return errors.New(str)
		} else {
			return nil
		}
	} else {
		return errors.New("closed")
	}
}
/************************************************ eprivate ************************************/
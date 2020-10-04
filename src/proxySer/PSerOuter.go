package proxySer

import (
	"context"
	"github.com/logs"
	"github.com/satori/go.uuid"
	"global"
	"net"
	"os"
	"strings"
	"sync"
)
//public
type PSerOuter struct {
	serAddr string
	logObj *logs.BeeLogger
	clientMap map[string] net.Conn
	inChanMsg, outChanMsg chan global.ChanDataParm
	listener *net.TCPListener
	exitCtx context.Context
	exitCancel context.CancelFunc
	lock sync.Mutex
}
func NewPSerOuter(serAddr string, logObj *logs.BeeLogger, inChanMsg, outChanMsg chan global.ChanDataParm) *PSerOuter {
	obj := &PSerOuter{serAddr:"0.0.0.0:" + strings.Split(serAddr, ":")[1], logObj:logObj, clientMap:make(map[string] net.Conn), inChanMsg:inChanMsg, outChanMsg:outChanMsg, listener:nil}
	obj.exitCtx, obj.exitCancel = context.WithCancel(context.Background())
	return obj
}
func (obj *PSerOuter) Run() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", obj.serAddr)
	if err != nil {
		obj.logObj.Error("net.ResovleTCPAddr fail:%s", obj.serAddr)
		os.Exit(0)
	}
	obj.listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		obj.logObj.Error("listen %s fail: %s", obj.serAddr, err)
		os.Exit(0)
	} else {
		obj.logObj.Debug("%s listen success", obj.serAddr)
	}
	go obj.listenChan()
	go func() {
		for {
			conn, err := obj.listener.Accept()
			if err != nil {
				obj.logObj.Error("[%s] accept happen error:%s", obj.serAddr, err.Error())
				break
			} else {
				obj.logObj.Debug("client [%s] connected in [%s]", conn.RemoteAddr().String(), obj.serAddr)
			}
			obj.newClient(conn)
		}
	}()
}
func (obj *PSerOuter) Stop() {
	if nil != obj.listener {
		obj.listener.Close()
		obj.listener = nil
	}
	obj.exitCancel()
}
//end public

//private
func (obj *PSerOuter) listenChan() {
	for {
		select {
		case msg := <- obj.inChanMsg:
			obj.handleMsg(&msg)
		case <- obj.exitCtx.Done():
			return
		}
	}
}
func (obj *PSerOuter) handleMsg(msg *global.ChanDataParm) {
	if nil == msg {
		return
	}
	if global.Closed == msg.CmdType {
		obj.delConn(msg.ClientId)
	} else if global.Connected == msg.CmdType {

	} else if global.Data == msg.CmdType {
		connObj := obj.getConnByClientId(msg.ClientId)
		if nil != connObj {
			slen, err_ := connObj.Write(msg.Data[:msg.DataLen])
			if nil != err_ {
				obj.logObj.Warning("send sub fail:%s", err_.Error())
			} else if msg.DataLen != slen {
				obj.logObj.Warning("slen(%d) != len(%d)", slen, msg.DataLen)
			} else {
				obj.logObj.Debug("W:%d", slen)
			}
		}
	}
}
func (obj *PSerOuter) getConnByClientId(clientId string) net.Conn {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	connObj, ok := obj.clientMap[clientId]
	if ok {
		return connObj
	} else {
		return nil
	}
}
func (obj *PSerOuter) storeConn(clientId string, connObj net.Conn) {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	obj.clientMap[clientId] = connObj
}
func (obj *PSerOuter) delConn(clientId string) {
	obj.lock.Lock()
	defer obj.lock.Unlock()
	connObj, ok := obj.clientMap[clientId]
	if ok {
		connObj.Close()
	}
	delete(obj.clientMap, clientId)
}
func (obj *PSerOuter) newClient(connObj net.Conn) {
	clientId := uuid.NewId()
	msg := global.ChanDataParm{CmdType:global.Connected, ClientId:clientId}
	obj.outChanMsg <- msg
	obj.storeConn(clientId, connObj)
	go obj.clientEvent(connObj, clientId)
}
func (obj *PSerOuter) clientEvent(connObj net.Conn, clientId string) {
	recvCacheBuf := make([]byte, global.OutClientRecvSize)
	ForEnd:
	for {
		rsize_, err_ := connObj.Read(recvCacheBuf[0:])
		if nil != err_ {
			obj.logObj.Debug("[%s] recv closed,err:%s", connObj.RemoteAddr().String(), err_.Error())
			break ForEnd
		} else if rsize_ > 0 {
			msg := global.ChanDataParm{CmdType:global.Data, ClientId:clientId}
			msg.DataLen = rsize_
			msg.Data = append(msg.Data, recvCacheBuf[0:rsize_]...)
			obj.outChanMsg <- msg
			obj.logObj.Debug("R:%d", rsize_)
		} else {
			obj.logObj.Debug("[%s:%d] recv closed", connObj.RemoteAddr().String(), rsize_)
			break ForEnd
		}
	}
	msg := global.ChanDataParm{CmdType:global.Closed, ClientId:clientId}
	obj.outChanMsg <- msg
}
//end private
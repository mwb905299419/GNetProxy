package proxySer

import (
	"github.com/logs"
	"global"
	"strings"
	"tcp"
)
/*
1字节(命令类型) + 1字节(clientId长度) + clientId + data(实际数据)
*/
//public
type PSerInner struct {
	logObj *logs.BeeLogger
	inChanMsg, outChanMsg chan global.ChanDataParm
	serObj *tcp.TcpSer
	exit chan bool
	sendBuf []byte
}
func NewPSerInner(serAddr string, logObj *logs.BeeLogger, inChanMsg, outChanMsg chan global.ChanDataParm) *PSerInner {
	obj := &PSerInner{logObj:logObj, inChanMsg:inChanMsg, outChanMsg:outChanMsg}
	obj.serObj = tcp.NewTcpSer(logObj, "0.0.0.0:" + strings.Split(serAddr, ":")[1], "PSerInner", obj.clientData)
	obj.exit = make(chan bool, 1)
	obj.sendBuf = make([]byte, global.MaxBufSize)
	return obj
}
func (obj *PSerInner) Run() {
	if nil == obj.serObj {
		return
	}
	go obj.listenChan()
	obj.serObj.Run()
}
func (obj *PSerInner) Stop() {
	if nil != obj.serObj {
		obj.serObj.Stop()
		obj.serObj = nil
	}
	obj.exit <- true
}
//end public

//private
func (obj *PSerInner) clientData(data []byte, dataLen int) {
	obj.outChanMsg <- global.ParseChanDataMsg(data, dataLen)
}
func (obj *PSerInner) listenChan() {
	for {
		select {
		case msg := <- obj.inChanMsg:
			obj.handleMsg(&msg)
		case exitTag := <- obj.exit:
			if exitTag {
				return
			}
		}
	}
}
func (obj *PSerInner) handleMsg(msg *global.ChanDataParm) {
	if nil == msg || nil == obj.serObj {
		return
	}
	obj.serObj.RwLock.RLock()
	defer obj.serObj.RwLock.RUnlock()
	sendLen := global.OrgChanDataMsg(msg.CmdType, msg.ClientId, msg.Data, msg.DataLen, obj.sendBuf)
	for _, client := range obj.serObj.ClientList {
		client.SendTTData(obj.sendBuf, sendLen)
		break
	}
}
//end private

package tcp

/*
1          + 1         + 4                          + 1    + 1        + Data     + 1
FRAME_HEAD + FRAME_VER + 从第8字节到末尾减1的字节数 + crc8 + 消息类型 + 消息数据 + FRAME_TAIL
*/
import (
	"github.com/logs"
	"net"
	"os"
	"sync"
	"time"
)

type TcpSer struct {
	listenAddr string
	logObj *logs.BeeLogger
	ClientList []*TcpClient
	RwLock *sync.RWMutex
	serName string
	clientDataCb func(data []byte, dataLen int)
	clientClosedCb func(clientId string)
	listener *net.TCPListener
	exit chan bool
}
/*************************************************** spublic **********************************************/
func NewTcpSer(log *logs.BeeLogger, listenAddr, serName string, clientDataCb func(data []byte, dataLen int)) *TcpSer {
	obj := &TcpSer{logObj:log, RwLock:new(sync.RWMutex), serName:serName, listenAddr:listenAddr, clientDataCb:clientDataCb,
		listener:nil, exit:make(chan bool, 1)}
	obj.clientClosedCb = nil
	return obj
}
func (obj *TcpSer) Run() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", obj.listenAddr)
	if err != nil {
		obj.logObj.Error("net.ResovleTCPAddr fail:%s", obj.listenAddr)
		os.Exit(0)
	}
	obj.listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		obj.logObj.Error("%s listen %s fail: %s", obj.serName, obj.listenAddr, err)
		os.Exit(0)
	} else {
		obj.logObj.Debug("%s %s listen success", obj.listenAddr, obj.serName)
	}

	go obj.timeOut()
	go func() {
		for {
			conn, err := obj.listener.Accept()
			if err != nil {
				obj.logObj.Error("%s [%s] accept happen error:%s", obj.serName, obj.listenAddr, err.Error())
				return
			} else {
				obj.logObj.Debug("client [%s] connected in %s [%s]", conn.RemoteAddr().String(), obj.serName, obj.listenAddr)
			}
			client_ := newTcpClient(conn, obj.logObj, obj.clientIdExists, obj.clientDataCb)
			obj.RwLock.Lock()
			obj.ClientList = append(obj.ClientList, client_)
			obj.RwLock.Unlock()
		}
	}()
}
func (obj *TcpSer) DelClient(clientId string) {
	obj.RwLock.Lock()
	defer obj.RwLock.Unlock()
	for index_ := 0; index_ < len(obj.ClientList); index_++ {
		v_ := obj.ClientList[index_]
		if clientId == v_.GetClientId() {
			v_.SendExit()
			v_.closeTcpClient()
			obj.ClientList = append(obj.ClientList[:index_], obj.ClientList[index_+1:]...)
			obj.logObj.Debug("%s DelClient [%s]", obj.serName, clientId)
		}
	}
}
func (obj *TcpSer) Stop() {
	if nil != obj.listener {
		obj.listener.Close()
	}
	obj.exit <- true
}
func (obj *TcpSer) SetClientClosedCb(cb func(clientId string)) {
	obj.clientClosedCb = cb
}
/*************************************************** epublic **********************************************/

/*************************************************** sprivate **********************************************/
func (obj *TcpSer) clientIdExists(clientId string) bool {
	obj.RwLock.RLock()
	defer obj.RwLock.RUnlock()
	for _, client := range obj.ClientList {
		if clientId == client.GetClientId() {
			return true
		}
	}
	return false
}
func (obj_ *TcpSer) timeOut() {
	for {
		select{
		case <- time.After(1 * time.Second):
			obj_.checkClient()
		case exitTag := <- obj_.exit:
			if exitTag {
				return
			}
		}
	}
}
func (obj *TcpSer) checkClient() {
	obj.RwLock.Lock()
	defer obj.RwLock.Unlock()
	for index_ := 0; index_ < len(obj.ClientList); {
		v_ := obj.ClientList[index_]
		if true == v_.isTimeOut(){
			if nil != obj.clientClosedCb {
				obj.clientClosedCb(v_.driveId)
			}
			v_.SendExit()
			v_.closeTcpClient()
			obj.ClientList = append(obj.ClientList[:index_], obj.ClientList[index_+1:]...)
			obj.logObj.Debug("%s CloseTcpClient [%s]", obj.serName, v_.connObj.RemoteAddr().String())
		} else if true == v_.invalid {
			if nil != obj.clientClosedCb {
				obj.clientClosedCb(v_.driveId)
			}
			obj.logObj.Debug("%s client [%s] is invalid", obj.serName, v_.connObj.RemoteAddr().String())
			obj.ClientList = append(obj.ClientList[:index_], obj.ClientList[index_+1:]...)
		} else {
			index_++
		}
	}
}
/*************************************************** eprivate **********************************************/

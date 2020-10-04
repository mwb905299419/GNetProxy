package global

const (
	InnerClientRecvSize = (1024 * 32) //内网接收最大缓冲区
	MaxBufSize = (1024 * 1024 * 8)
	OutClientRecvSize =(1024 * 32) //外网接收最大缓冲区
	MsgChanSize = 1024
	Connected = iota
	Closed
	Data
	ProxyClientLoginReq //关联结构体 ProxyClientLoginReqData
	ProxyClientLoginRep //关联结构体 ProxyClientLoginRepData
	QueryProxyClientInfo
	RepProxyClientInfo
	AddProxyClientInfo  //关联结构体 ProxyClientMapP
	DelProxyClientInfo  //关联结构体 ProxyClientMapP
	SyncProxyClientInfo //关联结构体 ProxyClientP
	SyncProxyClientExit //关联结构体 ProxyClientP
	DelProxyClientUser  //关联结构体 string
	MAddSerPair //关联结构体 SerPair
	MDelSerPair //关联结构体 SerPair
)

//ProxyClientLoginReq 数据格式：
type ProxyClientLoginReqData struct {
	UserName string
	MD5Str string //UserName:UserPwd 的 MD5 值
}
//ProxyClientLoginRep 数据格式：
type ProxyClientLoginRepData struct {
	Result int //0:失败,1:成功
}
//RepProxyClientInfo 数据格式：
type ProxyClientMapP struct {
	PSerInnerAddr string
	TerAddr string
	Desc string
}
type ProxyClientP struct {
	TerAddrMapList [] ProxyClientMapP
}
//AddProxyClientInfo 数据格式：ProxyClientMapP
//DelProxyClientInfo 数据格式：ProxyClientMapP

type ChanDataParm struct {
	CmdType byte
	ClientId string
	Data []byte
	DataLen int
}

type HttpToProxyCmdSerP struct {
	CmdType byte
	Data interface{}
}
//1字节(命令类型) + 1字节(clientId长度) + clientId + data(实际数据)
func ParseChanDataMsg(inputData []byte, inputDataLen int) ChanDataParm {
	msg := ChanDataParm{}
	msg.CmdType = inputData[0]
	idLen := inputData[1]
	msg.DataLen = inputDataLen - 2 - int(idLen)
	msg.ClientId = string(inputData[2:2 + idLen])
	msg.Data = append(msg.Data, inputData[2 + idLen:inputDataLen]...)
	return msg
}
func OrgChanDataMsg(cmdType byte, clientId string, inData []byte, inDataLen int, outData[]byte) int {
	outData[0] = cmdType
	outData[1] = byte(len(clientId))
	copy(outData[2:], []byte(clientId))
	copy(outData[2 + len(clientId):], inData[:inDataLen])
	return 2 + len(clientId) + inDataLen
}
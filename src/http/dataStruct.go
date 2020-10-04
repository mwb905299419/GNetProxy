package http

import (
	"github.com/json-iterator/go"
	"global"
)

const (
	Token = "Token"
	ContentTypeJson = "application/json"

	HAddUser = "/user/add" //添加用户，POST 方式，传入结构体 UserInfo
	HDelUser = "/user/del" //删除用户，GET方式，参数 /user/del?userName=''
	HUpdateUser = "/user/update" //更新用户信息，POST方式，传入结构体 UserInfo
	HQueryUserList = "/user/allUsers" //获取所有用户信息，GET方式，无参数，返回结构体 QueryAllUserRep

	HSetTransmit = "/transmit/set"   //配置转发信息，POST方式，传入结构体 TransmitSetP，返回结构体 TransmitSetResp
	HDelTransmit = "/transmit/del"	 //删除转发信息，POST方式，传入结构体 TransmitInfo，返回结构体 CommResponse
	HGetTransmit = "/transmit/get"	 //获取转发信息，GET方式，参数 /transmit/get?userName=''，返回结构体 TransmitGetResp
	HUpdateTransmit = "/transmit/update" //更新转发信息，POST方式，传入结构体 TransmitInfo，返回结构体 CommResponse
)

type CommResponse struct {
	StatusCode int  //1:成功,0:失败
	Status string	//对StatusCode的描述
}
type QueryAllUserRep struct {
	Result CommResponse
	ClientUserList []global.UserInfo
}
type TransmitSetP struct {
	UserName string
	TerAddr string
	Desc string
}
type TransmitInfo struct {
	PSerOuterAddr string
	TerAddr string
	Desc string
}
type TransmitSetResp struct {
	Result CommResponse
	TInfo TransmitInfo
}
type TransmitGetResp struct {
	Result CommResponse
	TransList []TransmitInfo
}

func getAllUserData() []byte {
	global.GConfig.UserLock.RLock()
	defer global.GConfig.UserLock.RUnlock()

	tmp := QueryAllUserRep{}
	tmp.Result.StatusCode = 1
	tmp.Result.Status = "OK"
	tmp.ClientUserList = global.GConfig.ClUserList.ClientUserList

	data, _ := jsoniter.Marshal(&tmp)
	return data
}
func getUserSerPair(userName string) []byte {
	global.GConfig.SerPairLock.Lock()
	defer global.GConfig.SerPairLock.Unlock()

	var retObj TransmitGetResp
	retObj.Result.Status = "OK"
	retObj.Result.StatusCode = 1
	for _, onePair := range global.GConfig.SerPairs.SerPairList {
		if onePair.UserName == userName {
			one := TransmitInfo{PSerOuterAddr:onePair.PSerOuterAddr, TerAddr:onePair.TerAddr, Desc:onePair.Desc}
			retObj.TransList = append(retObj.TransList, one)
		}
	}
	data,_ := jsoniter.Marshal(&retObj)
	return data
}


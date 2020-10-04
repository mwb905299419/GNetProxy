package global

import (
	"crypto/md5"
	"fmt"
	"github.com/config"
	"github.com/json-iterator/go"
	"github.com/kataras/iris/core/errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"units"
)

type LogConfig struct {
	LogMode int //0:"console",1:"file"
	LogLevel int //=7:Debug,Warning,Error;=4:Warning,Error;=3:Error;
}
type UserInfo struct {
	UserName, UserPwd string
	State int //0:不在线,1:在线
}
type SerPair struct {
	//对外暴露地址,对内通信地址,属于那个用户,对内目的访问地址,用途描述
	PSerOuterAddr, PSerInnerAddr, UserName, TerAddr, Desc string
}
type SerPairListP struct {
	SerPairList []SerPair
}
type ClientUserListP struct {
	ClientUserList []UserInfo
}
type ConfigGlobal struct {
	ClUserList ClientUserListP
	LogInfo LogConfig
	CmdSerAddr, CmdHttpSerAddr string
	SerPairs SerPairListP
	Token string
	UserLock sync.RWMutex
	SerPairLock sync.Mutex
}
var GConfig ConfigGlobal

var configPath = "config\\config.ini"
var userFile = "config\\proxyClientUser.json"
var serPairFile = "config\\proxySerPair.json"
func InitGlobalConfig(){
	useDebug := false
	if useDebug {
		configPath = "G:\\Lc\\program\\NetProxy\\bin\\config\\config.ini"
		userFile = "G:\\Lc\\program\\NetProxy\\bin\\config\\proxyClientUser.json"
		serPairFile = "G:\\Lc\\program\\NetProxy\\bin\\config\\proxySerPair.json"
	}
	initLocalConfig()
	initUserList()
	initSerPair()
}
//private
func initLocalConfig() {
	iniconf, err := config.NewConfig("ini", configPath)
	if err != nil {
		fmt.Printf("%s:%s\r\n","config.NewConfig [" + configPath + "] fail", err)
	} else {
		GConfig.CmdSerAddr = iniconf.String("CmdSerAddr")
		GConfig.Token = iniconf.String("Token")
		GConfig.CmdHttpSerAddr = iniconf.String("CmdHttpSerAddr")
		GConfig.LogInfo.LogMode,_ = iniconf.Int("LogInfo::LogMode")
		GConfig.LogInfo.LogLevel,_ = iniconf.Int("LogInfo::LogLevel")
	}
}
func initUserList() {
	f, err := os.OpenFile(userFile, os.O_RDONLY,0600)
	defer f.Close()
	if err !=nil {
	} else {
		contentByte,err := ioutil.ReadAll(f)
		if nil != err {
		} else {
			err = jsoniter.Unmarshal(contentByte, &GConfig.ClUserList)
			if nil != err {
				fmt.Println(err)
			} else {
				for index := 0; index < len(GConfig.ClUserList.ClientUserList); index++ {
					GConfig.ClUserList.ClientUserList[index].State = 0
				}
			}
		}
	}
}
func initSerPair() {
	f, err := os.OpenFile(serPairFile, os.O_RDONLY,0600)
	defer f.Close()
	if err !=nil {
	} else {
		contentByte,err := ioutil.ReadAll(f)
		if nil != err {
		} else {
			err = jsoniter.Unmarshal(contentByte, &GConfig.SerPairs)
			if nil != err {
				fmt.Println(err)
			}
		}
	}
}
func saveToFile(dataObj interface{}, path string) {
	data, err_ := jsoniter.Marshal(dataObj)
	if nil == err_ {
		f,err := os.Create(path)
		defer f.Close()
		if err !=nil {
			fmt.Println(err.Error())
		} else {
			_,err=f.Write([]byte(data))
			if nil != err {
				fmt.Println(err)
			}
		}
	}
}
func AddUser(oneUser UserInfo) error {
	GConfig.UserLock.Lock()
	defer GConfig.UserLock.Unlock()

	for _, user := range GConfig.ClUserList.ClientUserList {
		if user.UserName == oneUser.UserName {
			return errors.New("user already exists")
		}
	}
	GConfig.ClUserList.ClientUserList = append(GConfig.ClUserList.ClientUserList, oneUser)
	saveToFile(GConfig.ClUserList, userFile)
	return nil
}
func FindUser(userName string) bool {
	GConfig.UserLock.RLock()
	defer GConfig.UserLock.RUnlock()

	for _, user := range GConfig.ClUserList.ClientUserList {
		if user.UserName == userName {
			return true
		}
	}

	return false
}
func UpdateUser(oneUser UserInfo) error {
	GConfig.UserLock.Lock()
	defer GConfig.UserLock.Unlock()

	for index, user := range GConfig.ClUserList.ClientUserList {
		if user.UserName == oneUser.UserName {
			GConfig.ClUserList.ClientUserList[index].UserPwd = oneUser.UserPwd
			saveToFile(GConfig.ClUserList, userFile)
			return nil
		}
	}
	return errors.New("no find user")
}
func DelUser(userName string) error {
	GConfig.UserLock.Lock()
	defer GConfig.UserLock.Unlock()

	for index_ := 0; index_ < len(GConfig.ClUserList.ClientUserList); index_++ {
		if GConfig.ClUserList.ClientUserList[index_].UserName == userName {
			GConfig.ClUserList.ClientUserList = append(GConfig.ClUserList.ClientUserList[:index_], GConfig.ClUserList.ClientUserList[index_+1:]...)
			saveToFile(GConfig.ClUserList, userFile)
			DelUserSerPair(userName)
			return nil
		}
	}
	return errors.New("no find user")
}
func UserOffline(userName string) {
	GConfig.UserLock.RLock()
	defer GConfig.UserLock.RUnlock()

	for index := 0; index < len(GConfig.ClUserList.ClientUserList);index++ {
		if userName == GConfig.ClUserList.ClientUserList[index].UserName {
			GConfig.ClUserList.ClientUserList[index].State = 0
		}
	}
}
func UserAuth(userName, userMd5 string) bool {
	GConfig.UserLock.RLock()
	defer GConfig.UserLock.RUnlock()

	for index := 0; index < len(GConfig.ClUserList.ClientUserList);index++ {
		if userName == GConfig.ClUserList.ClientUserList[index].UserName {
			data := []byte(GConfig.ClUserList.ClientUserList[index].UserName + ":" + GConfig.ClUserList.ClientUserList[index].UserPwd)
			has := md5.Sum(data)
			tmp := fmt.Sprintf("%x", has)
			if userMd5 == tmp {
				GConfig.ClUserList.ClientUserList[index].State = 1
				return true
			} else {
				GConfig.ClUserList.ClientUserList[index].State = 0
				return false
			}
		}
	}
	return false
}
func AddSerPair(oneSerPair SerPair) error {
	GConfig.SerPairLock.Lock()
	defer GConfig.SerPairLock.Unlock()

	for _, onePair := range GConfig.SerPairs.SerPairList {
		if onePair.PSerInnerAddr == oneSerPair.PSerInnerAddr || onePair.PSerOuterAddr == oneSerPair.PSerOuterAddr {
			return errors.New("ser addr already exists")
		}
	}
	GConfig.SerPairs.SerPairList = append(GConfig.SerPairs.SerPairList, oneSerPair)
	saveToFile(GConfig.SerPairs, serPairFile)
	return nil
}
func DelSerPair(PSerOuterAddr string) (SerPair, error) {
	GConfig.SerPairLock.Lock()
	defer GConfig.SerPairLock.Unlock()

	oneSerPair := SerPair{}
	for index_ := 0; index_ < len(GConfig.SerPairs.SerPairList); index_++ {
		if GConfig.SerPairs.SerPairList[index_].PSerOuterAddr == PSerOuterAddr {
			oneSerPair.TerAddr = GConfig.SerPairs.SerPairList[index_].TerAddr
			oneSerPair.Desc = GConfig.SerPairs.SerPairList[index_].Desc
			oneSerPair.PSerOuterAddr = GConfig.SerPairs.SerPairList[index_].PSerOuterAddr
			oneSerPair.UserName = GConfig.SerPairs.SerPairList[index_].UserName
			oneSerPair.PSerInnerAddr = GConfig.SerPairs.SerPairList[index_].PSerInnerAddr
			GConfig.SerPairs.SerPairList = append(GConfig.SerPairs.SerPairList[:index_], GConfig.SerPairs.SerPairList[index_+1:]...)
			saveToFile(GConfig.SerPairs, serPairFile)
			return oneSerPair, nil
		}
	}
	return oneSerPair, errors.New("no find ser addr")
}
func DelUserSerPair(userName string) {
	GConfig.SerPairLock.Lock()
	defer GConfig.SerPairLock.Unlock()

	for index_ := 0; index_ < len(GConfig.SerPairs.SerPairList); {
		if GConfig.SerPairs.SerPairList[index_].UserName == userName {
			GConfig.SerPairs.SerPairList = append(GConfig.SerPairs.SerPairList[:index_], GConfig.SerPairs.SerPairList[index_+1:]...)
		} else {
			index_++
		}
	}
	saveToFile(GConfig.SerPairs, serPairFile)
}
func UpdateSerPair(PSerOuterAddr, Desc, TerAddr string) (SerPair,error) {
	GConfig.SerPairLock.Lock()
	defer GConfig.SerPairLock.Unlock()

	for index_ := 0; index_ < len(GConfig.SerPairs.SerPairList); index_++ {
		if GConfig.SerPairs.SerPairList[index_].PSerOuterAddr == PSerOuterAddr {
			GConfig.SerPairs.SerPairList[index_].Desc = Desc
			GConfig.SerPairs.SerPairList[index_].TerAddr = TerAddr
			saveToFile(GConfig.SerPairs, serPairFile)
			return GConfig.SerPairs.SerPairList[index_], nil
		}
	}
	return SerPair{}, errors.New("no find ser addr")
}
func GetPortPair() (string, string) {
	GConfig.SerPairLock.Lock()
	defer GConfig.SerPairLock.Unlock()

	var tryCount int = 200
	localIp := strings.Split(GConfig.CmdHttpSerAddr, ":")[0]
	for {
		outPort, _ := units.GetFreeTcpPort("0.0.0.0")
		outAddr := fmt.Sprintf("%s:%d", localIp,outPort)
		inPort, _ := units.GetFreeTcpPort("0.0.0.0")
		inAddr := fmt.Sprintf("%s:%d", localIp,inPort)
		if nil == GConfig.SerPairs.SerPairList || 0 == len(GConfig.SerPairs.SerPairList) {
			return outAddr, inAddr
		}
		for _, onePair := range GConfig.SerPairs.SerPairList {
			if onePair.PSerInnerAddr != inAddr && onePair.PSerOuterAddr != outAddr {
				return outAddr, inAddr
			}
		}
		tryCount--
		if tryCount < 0 {
			return "", ""
		}
	}
}
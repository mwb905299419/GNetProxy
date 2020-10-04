package proxyClient

import (
	"fmt"
	"github.com/config"
	"github.com/json-iterator/go"
	"github.com/kataras/iris/core/errors"
	"global"
	"io/ioutil"
	"os"
)

type RunParm struct {
	CmdSerAddr string
	UserName, UserPwd string
	LogMode, LogLevel int
}
var GConfig RunParm
var proxyClientTerAddrFile = "config/proxyClientTerAddr.json"
var configPath = "config\\proxyClientConfig.ini"
func getConfigInfo() error {
	useDebug := false
	if useDebug {
		proxyClientTerAddrFile = "G:\\Lc\\program\\GoProgram\\NetProxy\\bin\\config\\proxyClientTerAddr.json"
		configPath = "G:\\Lc\\program\\GoProgram\\NetProxy\\bin\\config\\proxyClientConfig.ini"
	}
	iniconf, err := config.NewConfig("ini", configPath)
	if err != nil {
		fmt.Printf("%s:%s\r\n","config.NewConfig [" + configPath + "] fail", err)
		return err
	} else {
		GConfig.CmdSerAddr = iniconf.String("CmdSerAddr")
		GConfig.LogMode,_ = iniconf.Int("LogInfo::LogMode")
		GConfig.LogLevel,_ = iniconf.Int("LogInfo::LogLevel")
		GConfig.UserName = iniconf.String("UserInfo::UserName")
		GConfig.UserPwd = iniconf.String("UserInfo::UserPwd")
		return nil
	}
}
func getProxyClientList(pcList *global.ProxyClientP){
	if nil == pcList {
		return
	}
	f, err := os.OpenFile(proxyClientTerAddrFile, os.O_RDONLY,0600)
	defer f.Close()
	if err !=nil {
	} else {
		contentByte,err := ioutil.ReadAll(f)
		if nil != err {
		} else {
			err = jsoniter.Unmarshal(contentByte, pcList)
			if nil != err {
				fmt.Println(err)
			}
		}
	}
}
func putProxyClientMap(cmap global.ProxyClientMapP, pcList *global.ProxyClientP) error {
	if nil == pcList {
		return errors.New("input nil")
	}
	for _, tmap := range pcList.TerAddrMapList {
		if tmap.PSerInnerAddr == cmap.PSerInnerAddr {//映射已经存在，不再添加
			return errors.New(tmap.PSerInnerAddr + " already exists")
		}
	}
	pcList.TerAddrMapList = append(pcList.TerAddrMapList, cmap)
	saveProxyClientMap(pcList)
	return nil
}
func delProxyClientMap(cmap global.ProxyClientMapP, pcList *global.ProxyClientP) {
	if nil == pcList {
		return
	}
	for index_ := 0; index_ < len(pcList.TerAddrMapList); index_++ {
		if pcList.TerAddrMapList[index_].PSerInnerAddr == cmap.PSerInnerAddr {
			pcList.TerAddrMapList = append(pcList.TerAddrMapList[:index_], pcList.TerAddrMapList[index_+1:]...)
			saveProxyClientMap(pcList)
			return
		}
	}
}
func saveProxyClientMap(pcList *global.ProxyClientP) {
	if nil == pcList {
		return
	}
	data, err_ := jsoniter.Marshal(pcList)
	if nil == err_ {
		f,err := os.Create(proxyClientTerAddrFile)
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


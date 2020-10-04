package units

import (
	"github.com/satori/go.uuid"
	"sync"
	"time"
)

type ChanData struct{
	C chan interface{}
	Used bool
	CTime int64
}
type ChanDataMap struct {
	lock sync.RWMutex
	chanMap map[string] *ChanData
}
func NewChanDataMap() *ChanDataMap {
	lobj := &ChanDataMap{chanMap:make(map[string]*ChanData)}
	return lobj
}
func (obj *ChanDataMap) GetChanById(keyId string) *ChanData {
	obj.lock.RLock()
	chanObj, ok := obj.chanMap[keyId]
	obj.lock.RUnlock()
	if ok {
		return chanObj
	} else {
		return nil
	}
}
func (obj *ChanDataMap) DelChanById(keyId string) {
	obj.lock.Lock()
	delete(obj.chanMap, keyId)
	obj.lock.Unlock()
}
func (obj *ChanDataMap) CreateChanById(keyId string) *ChanData {
	obj.lock.Lock()
	dataChan := &ChanData{C:make(chan interface{}, 1), Used:false, CTime: time.Now().Unix()}
	obj.chanMap[keyId] = dataChan
	obj.lock.Unlock()
	return dataChan
}
func (obj *ChanDataMap) CreateChan() (string, *ChanData) {
	obj.lock.Lock()
	dataChan := &ChanData{C:make(chan interface{}, 1), Used:false, CTime: time.Now().Unix()}
	keyId := uuid.NewId()
	obj.chanMap[keyId] = dataChan
	obj.lock.Unlock()
	return keyId, dataChan
}


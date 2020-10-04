package Queue

import (
    "github.com/logs"
    "global"
    "sync"
    "units"
)

var lock sync.Mutex

type Queen struct {
    Length   int64 //队列长度
    Capacity int64 //队列容量
    Head     int64 //队头
    Tail     int64 //队尾
    log_     *logs.BeeLogger
    Data     []interface{}
}

//初始化
func MakeQueen(length int64) Queen {
    var q = Queen{
        Length: length,
        log_ :  units.CreateLogobject("db", global.GConfig.StartParms.LogInfo.LogLevel, global.GConfig.StartParms.LogInfo.LogMode),
        Data:  make([]interface{}, length),
    }
    return q
}

//判断是否为空
func (t *Queen) IsEmpty() bool {
    return t.Capacity == 0
}

//判断是否满
func (t *Queen) IsFull() bool {
    return t.Capacity == t.Length
}

//加一个元素
func (t *Queen) Append(element interface{}) bool {
    if t.IsFull() {
        t.log_.Debug("队列已满 ，无法加入")
        return false
    }
    lock.Lock()
    t.Data[t.Tail] = element
    t.Tail++
    t.Capacity++
    lock.Unlock()
    return true
}

//弹出一个元素，并返回
func (t *Queen) OutElement() interface{} {
    if t.IsEmpty() {
        t.log_.Debug("队列为空，无法弹出")
        return nil
    }
    lock.Lock()
    t.Capacity--
    t.Head++
    lock.Unlock()
    return t.Data[t.Head]
}

//遍历
func (t *Queen) Each(fn func(node interface{})) {
    for i := t.Head; i < t.Head+t.Capacity; i++ {
        fn(t.Data[i%t.Length])
    }
}

//清空
func (t *Queen) Clcear() bool {
    lock.Lock()
    t.Capacity = 0
    t.Head = 0
    t.Tail = 0
    t.Data = make([]interface{}, t.Length)
    lock.Unlock()
    return true
}

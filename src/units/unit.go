package units

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kataras/iris/core/errors"
	"github.com/logs"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var crctable []byte = []byte{0, 94,188,226, 97, 63,221,131,194,156,126, 32,163,253, 31, 65,
157,195, 33,127,252,162, 64, 30, 95, 1,227,189, 62, 96,130,220,
35,125,159,193, 66, 28,254,160,225,191, 93, 3,128,222, 60, 98,
190,224, 2, 92,223,129, 99, 61,124, 34,192,158, 29, 67,161,255,
70, 24,250,164, 39,121,155,197,132,218, 56,102,229,187, 89, 7,
219,133,103, 57,186,228, 6, 88, 25, 71,165,251,120, 38,196,154,
101, 59,217,135, 4, 90,184,230,167,249, 27, 69,198,152,122, 36,
248,166, 68, 26,153,199, 37,123, 58,100,134,216, 91, 5,231,185,
140,210, 48,110,237,179, 81, 15, 78, 16,242,172, 47,113,147,205,
17, 79,173,243,112, 46,204,146,211,141,111, 49,178,236, 14, 80,
175,241, 19, 77,206,144,114, 44,109, 51,209,143, 12, 82,176,238,
50,108,142,208, 83, 13,239,177,240,174, 76, 18,145,207, 45,115,
202,148,118, 40,171,245, 23, 73, 8, 86,180,234,105, 55,213,139,
87, 9,235,181, 54,104,138,212,149,203, 41,119,244,170, 72, 22,
233,183, 85, 11,136,214, 52,106, 43,117,151,201, 74, 20,246,168,
116, 42,200,150, 21, 75,169,247,182,232, 10, 84,215,137,107, 53}

func Crc8(buffer_ []byte, len_ int) byte{
	var _crc byte = 0
	for i := 0; i < len_; i++ {
		_c := buffer_[i]
		_c ^= _crc
		_crc = crctable[_c]
	}
	return _crc
}
func BytesToIntS(b []byte) (int, error) {
	if len(b) == 3 {
		b = append([]byte{0},b...)
	}
	bytesBuffer := bytes.NewBuffer(b)
	switch len(b) {
	case 1:
		var tmp int8
		err := binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		return int(tmp), err
	case 2:
		var tmp int16
		err := binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		return int(tmp), err
	case 4:
		var tmp int32
		err := binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		return int(tmp), err
	default:
		return 0,fmt.Errorf("%s", "BytesToInt bytes lenth is invaild!")
	}
}
func Byte4ToUint32(bdata []byte) uint32 {
	return (uint32(bdata[0]) + (uint32(bdata[1]) << 8) + (uint32(bdata[2]) << 16) + (uint32(bdata[3]) << 24))
}
func CreateLogobject(moduleName string, logLevel, logMode int) *logs.BeeLogger {
	ret_ := logs.NewLogger()
	ret_.EnableFuncCallDepth(true)
	ret_.SetLevel(logLevel)
	if 1 == logMode {
		fileN := "./log/" + moduleName + "/" + time.Now().Format("20060102150405") + fmt.Sprintf("_%x", &ret_) + ".log"
		ret_.SetLogger(logs.AdapterFile,  `{"filename":"` + fileN + `"}`)
	} else {
		ret_.SetLogger(logs.AdapterConsole)
	}
	return ret_
}
func ForkProcess(exePath string, args ...string) error {
	cmd_ := exec.Command(exePath, args...)
	if nil == cmd_ {
		return errors.New("call exec command fail")
	}
	err_ := cmd_.Start()
	if nil != err_{
		return err_
	} else {
		return nil
	}
}
func ParseIpPort(addr string) (string, int) {
	//addr format is tcp://ip:port or ip:port
	arry := strings.Split(addr, "//")
	if 2 == len(arry) {
		arry = strings.Split(arry[1], ":")
	} else {
		arry = strings.Split(addr, ":")
	}
	port, _ := strconv.Atoi(arry[1])
	return arry[0], port
}
func GetFreeTcpPort(ip string) (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ip + ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
func SetOsEnv(kvMap map[string]string) error {
	for k,v := range kvMap {
		err := os.Setenv(k, v)
		if nil != err {
			return err
		}
	}
	return nil
}
func GetLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}

		}
	}
	return "", errors.New("no ip")
}
//remoteAddr format is ip:port
func GetLocalIpByRemote(remoteAddr string) (string, error) {
	conn, err_ := net.Dial("tcp", remoteAddr)
	if nil != err_ {
		return "", err_
	} else {
		arr := strings.Split(conn.LocalAddr().String(), ":")
		conn.Close()
		if len(arr) < 2 {
			return "", errors.New(conn.LocalAddr().String() + " not contain :")
		}
		return arr[0], nil
	}
}

package main

import (
	"fmt"
	"proxyClient"
	"sync"
)
func main() {
	fmt.Println("Hello NetProxyClient")
	var wait sync.WaitGroup
	wait.Add(1)

	proxyClientCmdObj := proxyClient.NewProxyClientCmd()
	if nil != proxyClientCmdObj {
		proxyClientCmdObj.Run()
	}

	wait.Wait()
}

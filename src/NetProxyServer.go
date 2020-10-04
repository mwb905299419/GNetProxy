package main

import (
	"fmt"
	"starter"
	"sync"
)

func main() {
	fmt.Println("Hello NetProxyServer")
	var wait sync.WaitGroup
	wait.Add(1)

	proxySerStarter := starter.NewStarter()
	proxySerStarter.Run()

	wait.Wait()
}

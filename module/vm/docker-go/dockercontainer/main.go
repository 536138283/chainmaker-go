package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"fmt"
	"time"
)

func main() {

	manager := module.NewManager()
	manager.InitContainer()

	// infinite loop
	// todo wait node send stop
	for i := 0; ; i++ {
		fmt.Println("in main process -- ", i)
		time.Sleep(time.Minute)
	}
}

package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"fmt"
	"log"
	"time"
)

func main() {

	manager := module.NewManager()
	err := manager.InitContainer()
	if err != nil {
		log.Fatalln(err)
	}

	// infinite loop
	for i := 0; ; i++ {
		fmt.Println("in main process -- ", i)
		time.Sleep(time.Minute)
	}
}

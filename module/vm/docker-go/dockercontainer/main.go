package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"fmt"
	"log"
	"time"
)

func main() {

	manager, err := module.NewManager()
	if err != nil {
		log.Fatalf("Err in creating manager: %s", err)
	}

	manager.InitContainer()

	// infinite loop
	// todo wait node send stop
	for i := 0; ; i++ {
		fmt.Println("in main process -- ", i)
		time.Sleep(time.Minute)
	}
}

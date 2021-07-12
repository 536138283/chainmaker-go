package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontroller"
)

func main() {
	dockercontroller.NewDockerManager("CHAIN1")
}

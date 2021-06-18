package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"fmt"
)

type SandBox struct {
}

func InitSandboxEnv() error {
	if err := SetTmpMod(); err != nil {
		return err
	}

	fmt.Println("Successfully set tmp file mod")

	if err := SetCGroup(); err != nil {
		return err
	}
	fmt.Println("Successfully set cgroup")

	if err := CreateNewUsers(config.UserNum); err != nil {
		return err
	}
	fmt.Println("Successfully create new users")
	fmt.Println("Init Sandbox Completed")
	return nil
}

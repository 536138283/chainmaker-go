package security

import (
	"fmt"
	"os"
)

type SandBox struct {
}

func InitSecurityEnv() error {
	if err := SetTmpMod(); err != nil {
		return err
	}

	fmt.Println("Successfully set tmp file mod")

	if err := SetCGroup(); err != nil {
		return err
	}
	fmt.Println("Successfully set cgroup")

	fmt.Println("Init Sandbox Completed")
	return nil
}

func SetTmpMod() error {
	return os.Chmod("/tmp/", 0777)
}

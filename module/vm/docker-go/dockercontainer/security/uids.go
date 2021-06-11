package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"os"
)

type Users struct {
}

// CreateNewUsers create new users in docker from 10000 as uid,
func CreateNewUsers(userNum int) error {
	const AddUserFormat = "useradd -u %d -d /home/%s -m -s /bin/bash %s"
	const BaseUid = 10000

	for i := 0; i < userNum; i++ {
		newUserId := BaseUid + i
		newUserName := fmt.Sprintf("user%d", newUserId)
		addUserCommand := fmt.Sprintf(AddUserFormat, newUserId, newUserName, newUserName)

		if err := utils.RunCmd(addUserCommand); err != nil {
			return err
		}

		if err := setUserDirMod(newUserName); err != nil {
			return nil
		}

	}

	return nil

}

func SetTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}

// change userDir mod as 700
func setUserDirMod(newUserName string) error {
	newUserDir := fmt.Sprintf("/home/%s", newUserName)
	return os.Chmod(newUserDir, 0700)
}

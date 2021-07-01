package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type UsersController struct {
	Total   int
	UserMap map[int]*helper.User
	logger  *zap.SugaredLogger
}

func NewUsersController() *UsersController {

	userMap := make(map[int]*helper.User)

	users := &UsersController{
		UserMap: userMap,
		Total:   0,
		logger:  logger.NewDockerLogger(logger.MODULE_USERCONTROLLER),
	}

	return users
}

// CreateNewUsers create new users in docker from 10000 as uid,
func (u *UsersController) CreateNewUsers(userNum int) error {
	const AddUserFormat = "useradd -u %d -d /home/%s -m -s /bin/bash %s"
	const BaseUid = 10000

	for i := 0; i < userNum; i++ {
		newUserId := BaseUid + i
		newUser := u.createNewUser(newUserId)
		addUserCommand := fmt.Sprintf(AddUserFormat, newUserId, newUser.UserName, newUser.UserName)

		if err := utils.RunCmd(addUserCommand); err != nil {
			return err
		}

		if err := u.setUserDirMod(*newUser); err != nil {
			return nil
		}

		// update Users
		u.UserMap[newUserId] = newUser
		u.Total++

	}

	u.logger.Infof("create [%d] users", userNum)

	return nil
}

func (u *UsersController) createNewUser(userId int) *helper.User {

	const UserHomePath = "/home/u-%d"
	userName := fmt.Sprintf("u-%d", userId)
	binFileName := fmt.Sprintf("u-%d", userId)

	homeDir := fmt.Sprintf(UserHomePath, userId)
	sockPath := config.SockPath
	binPath := filepath.Join(homeDir, binFileName)

	return &helper.User{
		Uid:      userId,
		Gid:      userId,
		UserName: userName,
		HomeDir:  homeDir,
		SockPath: sockPath,
		BinPath:  binPath,
		Busy:     false,
	}
}

// change userDir mod as 700
func (u *UsersController) setUserDirMod(newUser helper.User) error {
	return os.Chmod(newUser.HomeDir, 0700)
}

func (u *UsersController) UpdateUserState(userId int, busy bool) {
	u.UserMap[userId].Busy = busy
	u.logger.Debugf("update user: [%v]", u.UserMap[userId])
}

func (u *UsersController) GetAvailableUser() *helper.User {
	for _, user := range u.UserMap {
		if !user.Busy {
			u.logger.Debugf("allocate user: [%v]", user)
			return user
		}
	}

	return nil
}

func (u *UsersController) ResetUserEnv(user *helper.User) error {
	rmCommand := fmt.Sprintf("rm -rf %s/*", user.HomeDir)
	u.logger.Debugf("reset user [%s] environment\n", user.UserName)
	return utils.RunCmd(rmCommand)
}

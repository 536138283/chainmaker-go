package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"github.com/enriquebris/goconcurrentqueue"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type UsersController struct {
	lock sync.RWMutex

	userQueue *goconcurrentqueue.FixedFIFO
	logger    *zap.SugaredLogger
}

func NewUsersController() *UsersController {

	//userMap := make(map[int]*helper.User)
	userQueue := goconcurrentqueue.NewFixedFIFO(config.UserNum)

	users := &UsersController{
		lock:      sync.RWMutex{},
		userQueue: userQueue,
		logger:    logger.NewDockerLogger(logger.MODULE_USERCONTROLLER),
	}

	return users
}

// CreateNewUsers create new users in docker from 10000 as uid,
func (u *UsersController) CreateNewUsers(userNum int) error {

	startTime := time.Now()
	const BaseUid = 10000

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < userNum/10; j++ {
				newUserId := BaseUid + i*userNum/10 + j
				u.generateNewUser(newUserId)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	u.logger.Infof("create [%d] users", userNum)
	u.logger.Infof("created user time: [%s]", time.Since(startTime))

	return nil
}

func (u *UsersController) generateNewUser(newUserId int) error {

	const AddUserFormat = "useradd -u %d -d /home/%s -m -s /bin/bash %s"

	newUser := u.constructNewUser(newUserId)
	addUserCommand := fmt.Sprintf(AddUserFormat, newUserId, newUser.UserName, newUser.UserName)

	if err := utils.RunCmd(addUserCommand); err != nil {
		return err
	}

	if err := u.setUserDirMod(*newUser); err != nil {
		return nil
	}

	// add created user to queue
	err := u.userQueue.Enqueue(newUser)
	if err != nil {
		return err
	}
	return nil
}

func (u *UsersController) constructNewUser(userId int) *helper.User {

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
	}
}

// change userDir mod as 700
func (u *UsersController) setUserDirMod(newUser helper.User) error {
	return os.Chmod(newUser.HomeDir, 0700)
}

//func (u *UsersController) UpdateUserState(userId int, busy bool) {
//	u.lock.Lock()
//	defer u.lock.Unlock()
//	u.UserMap[userId].Busy = busy
//	u.logger.Debugf("update user: [%v]", u.UserMap[userId])
//}

// GetAvailableUser pop user from queue header
func (u *UsersController) GetAvailableUser() (*helper.User, error) {

	user, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		return nil, err
	}

	u.logger.Debugf("get avaiable user: [%v]", user)
	return user.(*helper.User), nil
}

// FreeUser add user to queue tail
func (u *UsersController) FreeUser(user *helper.User) error {
	err := u.userQueue.Enqueue(user)
	if err != nil {
		return err
	}
	u.logger.Debugf("free user: [%v]", user)
	return nil
}

func (u *UsersController) ResetUserEnv(user *helper.User) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	rmCommand := fmt.Sprintf("rm -rf %s/*", user.HomeDir)
	u.logger.Debugf("reset user [%s] environment\n", user.UserName)
	return utils.RunCmd(rmCommand)
}

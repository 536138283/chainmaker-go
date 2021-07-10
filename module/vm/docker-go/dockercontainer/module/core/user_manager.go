package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"github.com/enriquebris/goconcurrentqueue"
	"go.uber.org/zap"
	"sync"
	"time"
)

type UsersManager struct {
	userQueue *goconcurrentqueue.FixedFIFO
	logger    *zap.SugaredLogger
}

func NewUsersManager() *UsersManager {

	userQueue := goconcurrentqueue.NewFixedFIFO(config.UserNum)

	usersManager := &UsersManager{
		userQueue: userQueue,
		logger:    logger.NewDockerLogger(logger.MODULE_USERCONTROLLER),
	}

	return usersManager
}

// CreateNewUsers create new users in docker from 10000 as uid
func (u *UsersManager) CreateNewUsers(userNum int) error {

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

func (u *UsersManager) generateNewUser(newUserId int) error {

	const AddUserFormat = "useradd -u %d %s"

	newUser := u.constructNewUser(newUserId)
	addUserCommand := fmt.Sprintf(AddUserFormat, newUserId, newUser.UserName)

	if err := utils.RunCmd(addUserCommand); err != nil {
		return err
	}

	// add created user to queue
	err := u.userQueue.Enqueue(newUser)
	if err != nil {
		return err
	}
	return nil
}

func (u *UsersManager) constructNewUser(userId int) *helper.User {

	userName := fmt.Sprintf("u-%d", userId)
	sockPath := config.SockPath

	return &helper.User{
		Uid:      userId,
		Gid:      userId,
		UserName: userName,
		SockPath: sockPath,
	}
}

// GetAvailableUser pop user from queue header
func (u *UsersManager) GetAvailableUser() (*helper.User, error) {

	user, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		return nil, err
	}

	u.logger.Debugf("get avaiable user: [%v]", user)
	return user.(*helper.User), nil
}

// FreeUser add user to queue tail
func (u *UsersManager) FreeUser(user *helper.User) error {
	err := u.userQueue.Enqueue(user)
	if err != nil {
		return err
	}
	u.logger.Debugf("free user: [%v]", user)
	return nil
}

package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type UsersManager struct {
	userQueue *utils.FixedFIFO
	logger    *zap.SugaredLogger
	userNum   int
}

func NewUsersManager() *UsersManager {

	userNumConfig := os.Getenv("UserNum")
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	userQueue := utils.NewFixedFIFO(userNum)

	usersManager := &UsersManager{
		userQueue: userQueue,
		logger:    logger.NewDockerLogger(logger.MODULE_USERCONTROLLER),
		userNum:   userNum,
	}

	return usersManager
}

// CreateNewUsers create new users in docker from 10000 as uid
func (u *UsersManager) CreateNewUsers() error {

	var err error

	startTime := time.Now()
	const BaseUid = 10000

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < u.userNum/10; j++ {
				newUserId := BaseUid + i*u.userNum/10 + j
				err = u.generateNewUser(newUserId)

				if err != nil {
					u.logger.Errorf("fail to create user [%d]", newUserId)
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	u.logger.Infof("create [%d] users", u.userNum)
	u.logger.Infof("created user time: [%s]", time.Since(startTime))

	return nil
}

func (u *UsersManager) generateNewUser(newUserId int) error {

	const AddUserFormat = "useradd -u %d %s"

	newUser := u.constructNewUser(newUserId)
	addUserCommand := fmt.Sprintf(AddUserFormat, newUserId, newUser.UserName)

	if err := utils.RunCmd(addUserCommand); err != nil {
		u.logger.Errorf("fail to run cmd : [%s]",addUserCommand)
		return err
	}

	// add created user to queue
	err := u.userQueue.Enqueue(newUser)
	if err != nil {
		u.logger.Errorf("fail to add created user to queue, newUser : [%v]",newUser)
		return err
	}
	return nil
}

func (u *UsersManager) constructNewUser(userId int) *security.User {

	userName := fmt.Sprintf("u-%d", userId)
	sockPath := filepath.Join(config.DMSDir, config.DMSSockPath)

	return &security.User{
		Uid:      userId,
		Gid:      userId,
		UserName: userName,
		SockPath: sockPath,
	}
}

// GetAvailableUser pop user from queue header
func (u *UsersManager) GetAvailableUser() (*security.User, error) {

	user, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		u.logger.Errorf("fail to call DequeueOrWaitForNextElement")
		return nil, err
	}

	u.logger.Debugf("get avaiable user: [%v]", user)
	return user.(*security.User), nil
}

// FreeUser add user to queue tail
func (u *UsersManager) FreeUser(user *security.User) error {
	err := u.userQueue.Enqueue(user)
	if err != nil {
		u.logger.Errorf("fail to call Enqueue")
		return err
	}
	u.logger.Debugf("free user: [%v]", user)
	return nil
}

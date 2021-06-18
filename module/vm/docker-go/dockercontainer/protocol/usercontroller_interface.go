package protocol

import security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"

type UserController interface {
	CreateNewUsers(userNum int) error

	GetAvailableUser() *security2.User

	UpdateUserState(userId int, busy bool)

	ResetUserEnv(user *security2.User) error
}

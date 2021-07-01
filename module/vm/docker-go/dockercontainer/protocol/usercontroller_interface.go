package protocol

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
)

type UserController interface {
	GetAvailableUser() *helper.User

	UpdateUserState(userId int, busy bool)

	ResetUserEnv(user *helper.User) error
}

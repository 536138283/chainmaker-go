package protocol

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
)

type UserController interface {
	GetAvailableUser() (*helper.User, error)

	FreeUser(user *helper.User) error
}

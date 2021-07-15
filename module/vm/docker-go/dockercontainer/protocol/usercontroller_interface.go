package protocol

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
)

type UserController interface {
	GetAvailableUser() (*security.User, error)

	FreeUser(user *security.User) error
}

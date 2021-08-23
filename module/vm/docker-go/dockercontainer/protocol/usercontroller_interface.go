/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
)

type UserController interface {
	// GetAvailableUser get available user
	GetAvailableUser() (*security.User, error)

	// FreeUser free user
	FreeUser(user *security.User) error
}

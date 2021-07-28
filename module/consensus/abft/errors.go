/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import "errors"

var (
	ErrDuplicatedRBCRequest = errors.New("receive duplicated reqeust")
)

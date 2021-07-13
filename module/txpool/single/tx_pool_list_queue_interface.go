/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

type queue interface {
	Add(key string, val interface{}) bool
	Get(key string) interface{}
	Size() int
	Remove(key string) (bool, interface{})
}

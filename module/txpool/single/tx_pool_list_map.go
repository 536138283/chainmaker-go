/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

type randomMap struct {
	m map[string]interface{}
}

func newRandomMap() *randomMap {
	return &randomMap{
		m: make(map[string]interface{}),
	}
}

func (listMap *randomMap) Add(key string, val interface{}) bool {

	if _, isExists := listMap.m[key]; isExists {
		return false
	}

	listMap.m[key] = val
	return true
}

func (listMap *randomMap) Get(key string) interface{} {
	val, isExists := listMap.m[key]
	if !isExists {
		return nil
	}
	return val
}

func (listMap *randomMap) Size() int {
	return len(listMap.m)
}

func (listMap *randomMap) Remove(key string) (bool, interface{}) {
	val, isExists := listMap.m[key]
	if !isExists {
		return false, nil
	}

	delete(listMap.m, key)
	return true, val
}

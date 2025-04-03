/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
)

const (
	stringType = "string"
	// solidity 全部int类型
	intType    = "int"
	int8Type   = "int8"
	int16Type  = "int16"
	int24Type  = "int24"
	int32Type  = "int32"
	int40Type  = "int40"
	int48Type  = "int48"
	int56Type  = "int56"
	int64Type  = "int64"
	int72Type  = "int72"
	int80Type  = "int80"
	int88Type  = "int88"
	int96Type  = "int96"
	int104Type = "int114"
	int112Type = "int112"
	int120Type = "int120"
	int128Type = "int128"
	int136Type = "int136"
	int144Type = "int144"
	int152Type = "int152"
	int160Type = "int160"
	int168Type = "int168"
	int176Type = "int176"
	int184Type = "int184"
	int192Type = "int192"
	int200Type = "int200"
	int208Type = "int208"
	int216Type = "int216"
	int224Type = "int224"
	int232Type = "int232"
	int240Type = "int240"
	int248Type = "int248"
	int256Type = "int256"
	// 全部uint类型
	uintType    = "uint"
	uint8Type   = "uint8"
	uint16Type  = "uint16"
	uint24Type  = "uint24"
	uint32Type  = "uint32"
	uint40Type  = "uint40"
	uint48Type  = "uint48"
	uint56Type  = "uint56"
	uint64Type  = "uint64"
	uint72Type  = "uint72"
	uint80Type  = "uint80"
	uint88Type  = "uint88"
	uint96Type  = "uint96"
	uint104Type = "uint114"
	uint112Type = "uint112"
	uint120Type = "uint120"
	uint128Type = "uint128"
	uint136Type = "uint136"
	uint144Type = "uint144"
	uint152Type = "uint152"
	uint160Type = "uint160"
	uint168Type = "uint168"
	uint176Type = "uint176"
	uint184Type = "uint184"
	uint192Type = "uint192"
	uint200Type = "uint200"
	uint208Type = "uint208"
	uint216Type = "uint216"
	uint224Type = "uint224"
	uint232Type = "uint232"
	uint240Type = "uint240"
	uint248Type = "uint248"
	uint256Type = "uint256"
	boolType    = "bool"
	addressType = "address"
	// solidity中的bytes类型
	bytesType   = "bytes"
	bytes1Type  = "bytes1"
	bytes2Type  = "bytes2"
	bytes3Type  = "bytes3"
	bytes4Type  = "bytes4"
	bytes5Type  = "bytes5"
	bytes6Type  = "bytes6"
	bytes7Type  = "bytes7"
	bytes8Type  = "bytes8"
	bytes9Type  = "bytes9"
	bytes10Type = "bytes10"
	bytes11Type = "bytes11"
	bytes12Type = "bytes12"
	bytes13Type = "bytes13"
	bytes14Type = "bytes14"
	bytes15Type = "bytes15"
	bytes16Type = "bytes16"
	bytes17Type = "bytes17"
	bytes18Type = "bytes18"
	bytes19Type = "bytes19"
	bytes20Type = "bytes20"
	bytes21Type = "bytes21"
	bytes22Type = "bytes22"
	bytes23Type = "bytes23"
	bytes24Type = "bytes24"
	bytes25Type = "bytes25"
	bytes26Type = "bytes26"
	bytes27Type = "bytes27"
	bytes28Type = "bytes28"
	bytes29Type = "bytes29"
	bytes30Type = "bytes30"
	bytes31Type = "bytes31"
	bytes32Type = "bytes32"
)

// parse 把interface{}类型，解析成为solidity类型中对应的go的类型
func parse(sType string, value interface{}) (arg interface{}, err error) {
	// 将solidity分割成两部分 以string[8]为例子 分割成 string [8]
	typeRegex := regexp.MustCompile(`^([a-zA-Z]+[0-9]*)((?:\[[0-9]*\])*)$`)
	matches := typeRegex.FindStringSubmatch(sType)
	if matches == nil {
		return nil, fmt.Errorf("invalid type format: %s", sType)
	}
	arrayPart := matches[2]
	// 处理非数组类型
	if arrayPart == "" {
		return baseTypeParse(sType, value)
	} else {
		// 数组类型
		arrayType := matches[1]
		// 处理数组类型
		arrayRegex := regexp.MustCompile(`\[([0-9]*)\]`)
		arrayMatches := arrayRegex.FindStringSubmatch(arrayPart)
		N, err := strconv.Atoi(arrayMatches[1])
		if err != nil {
			return nil, err
		}
		return parseArray(arrayType, value, N)
	}
}

// baseTypeParse 基本数据类型解析
func baseTypeParse(sType string, value interface{}) (interface{}, error) {
	switch sType {
	case stringType:
		return parseStr(value), nil
	case intType, int8Type, int16Type, int24Type, int32Type, int40Type, int48Type, int56Type, int64Type, int72Type,
		int80Type, int88Type, int96Type, int104Type, int112Type, int120Type, int128Type, int136Type, int144Type,
		int152Type, int160Type, int168Type, int176Type, int184Type, int192Type, int200Type, int208Type, int216Type,
		int224Type, int232Type, int240Type, int248Type, int256Type:
		return parseInt(sType, value)
	case uintType, uint8Type, uint16Type, uint24Type, uint32Type, uint40Type, uint48Type, uint56Type, uint64Type,
		uint72Type, uint80Type, uint88Type, uint96Type, uint104Type, uint112Type, uint120Type, uint128Type, uint136Type,
		uint144Type, uint152Type, uint160Type, uint168Type, uint176Type, uint184Type, uint192Type, uint200Type,
		uint224Type, uint232Type, uint240Type, uint248Type, uint256Type, uint208Type, uint216Type:
		return parseUInt(sType, value)
	case boolType:
		return parseBool(value)
	case addressType:
		return parseAddress(value), nil
	case bytesType:
		return parseBytes(sType, value), nil
	default:
		return value, nil
	}
}

// Param list
type Param map[string]interface{}

// loadFromJSON string into ABI data
func loadFromJSON(jString string) ([]Param, error) {
	if len(jString) == 0 {
		return nil, nil
	}
	data := []Param{}
	err := json.Unmarshal([]byte(jString), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Pack data into bytes
func Pack(a *abi.ABI, method string, paramsJson string) ([]byte, error) {
	param, err := loadFromJSON(paramsJson)
	if err != nil {
		return nil, err
	}

	var args []interface{}
	for _, p := range param {
		for k, v := range p {
			arg, err := parse(k, v)
			if err != nil {
				return nil, err
			}
			fmt.Println("qqqq", reflect.TypeOf(arg))
			args = append(args, arg)
		}
	}
	fmt.Println(args)
	return nil, nil
}

// parseInt 处理int类型数据
func parseInt(key string, value interface{}) (interface{}, error) {
	// solidity 不支持浮点数所以直接转换成int类型
	isM := regexp.MustCompile("(int)([0-9]+)")
	bitStr := isM.FindStringSubmatch(key)
	// 先把value转换成string类型
	valueStr := fmt.Sprint(value)
	if bitStr != nil {
		bitNum, err := strconv.ParseInt(bitStr[2], 10, 32)
		if err != nil {
			panic(fmt.Sprintf("parse int err: %s", err.Error()))
		}
		var result interface{}
		switch {
		case bitNum <= 8:
			num, err := strconv.ParseInt(valueStr, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int8: %s", err.Error())
			}
			result = int8(num)
		case bitNum <= 16:
			num, err := strconv.ParseInt(valueStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int16: %s", err.Error())
			}
			result = int16(num)
		case bitNum <= 32:
			num, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int32: %s", err.Error())
			}
			result = int32(num)
		case bitNum <= 64:
			num, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int64: %s", err.Error())
			}
			result = int64(num)
		default:
			// 对于大于 64 位的整数，使用 math/big 包
			bigInt := new(big.Int)
			num, ok := bigInt.SetString(valueStr, 10)
			if !ok {
				return nil, fmt.Errorf("failed to set big.Int from value: %s", valueStr)
			}
			result = num
		}
		return result, nil
	} else {
		// 如果 key 不是有效的 int 类型，尝试将其转换为 big.Int
		bigInt := new(big.Int)
		num, ok := bigInt.SetString(valueStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to set big.Int from value: %s", valueStr)
		}
		return num, nil
	}
}

// parseUInt 处理uint类型数据
func parseUInt(key string, value interface{}) (interface{}, error) {
	// solidity 不支持浮点数所以直接转换成uint类型
	isM := regexp.MustCompile("(uint)([0-9]+)")
	bitStr := isM.FindStringSubmatch(key)
	// 先把value转换成string类型
	valueStr := fmt.Sprint(value)
	if bitStr != nil {
		bitNum, err := strconv.ParseUint(bitStr[2], 10, 32)
		if err != nil {
			panic(fmt.Sprintf("parse int err: %s", err.Error()))
		}
		var result interface{}
		switch {
		case bitNum <= 8:
			num, err := strconv.ParseUint(valueStr, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int8: %s", err.Error())
			}
			result = uint8(num)
		case bitNum <= 16:
			num, err := strconv.ParseUint(valueStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int16: %s", err.Error())
			}
			result = uint16(num)
		case bitNum <= 32:
			num, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int32: %s", err.Error())
			}
			result = uint32(num)
		case bitNum <= 64:
			num, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int64: %s", err.Error())
			}
			result = uint64(num)
		default:
			// 对于大于 64 位的整数，使用 math/big 包
			bigInt := new(big.Int)
			num, ok := bigInt.SetString(valueStr, 10)
			if !ok {
				return nil, fmt.Errorf("failed to set big.Int from value: %s", valueStr)
			}
			result = num
		}
		return result, nil
	} else {
		// 如果 key 不是有效的 int 类型，尝试将其转换为 big.Int
		bigInt := new(big.Int)
		num, ok := bigInt.SetString(valueStr, 10)
		if !ok {
			return nil, fmt.Errorf("failed to set big.Int from value: %s", valueStr)
		}
		return num, nil
	}
}

// 处理string类型数据
func parseStr(value interface{}) string {
	return fmt.Sprint(value)
}

// 处理布尔类型数据
func parseBool(value interface{}) (interface{}, error) {
	v, ok := value.(bool)
	if !ok {
		fmt.Printf("value %v is not bool\n", value)
	}
	return v, nil
}

func parseAddress(value interface{}) interface{} {
	sAddress := fmt.Sprint(value)
	bytes := make([]byte, 0)
	bytes = append(bytes, []byte(sAddress)...)
	return bytes
}

func parseBytes(key string, value interface{}) (interface{}, error) {
	rest := key[len("bytes"):]

	sAddress := fmt.Sprint(value)
	// bytes为空代表动态的byte数组否则为定长byte数组
	if rest == "" {
		bytes := make([]byte, 0)

		bytes = append(bytes, []byte(sAddress)...)
		return bytes, nil
	}
	// 将剩余部分转换为整数
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", rest)
	}
	// 检查 N 是否在 1 到 32 之间
	if n < 1 || n > 32 {
		return 0, fmt.Errorf("bytes number must be between 1 and 32")
	}
	// 创建定长切片并返回
	bytes := make([]byte, n)
	for i := range []byte(sAddress) {
		bytes[i] = []byte(sAddress)[i]
	}
	return bytes, nil
}

// 解析int类型数组
func parseArray(key string, value interface{}, N int) (interface{}, error) {
	slice, ok := value.([]interface{})
	if !ok {
		return nil, errors.New("value is not a []interface{}")
	}
	if N == 0 {
		array := make([]interface{}, 0)
		for _, v := range slice {
			parseV, err := baseTypeParse(key, v)
			if err != nil {
				return nil, err
			}
			array = append(array, parseV)
		}
		return array, nil
	} else {
		array := make([]interface{}, 0)
		for i, v := range slice {
			parseV, err := parse(key, v)
			if err != nil {
				return nil, err
			}
			array[i] = parseV
		}
		return array, nil
	}
}

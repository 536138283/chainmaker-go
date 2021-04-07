/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"regexp"
)

func ErrResult(result *commonpb.Result, err error) (*commonpb.Result, error) {
	result.ContractResult.Message = err.Error()
	result.Code = commonpb.TxStatusCode_INVALID_PARAMETER
	result.ContractResult.Code = commonpb.ContractResultCode_FAIL
	return result, err
}

func runVM(tx *commonpb.Transaction, txSimContext protocol.TxSimContext, vmManager protocol.VmManager, log *logger.CMLogger) (*commonpb.Result, error) {
	var contractId *commonpb.ContractId
	var contractName string
	var runtimeType commonpb.RuntimeType
	var contractVersion string
	var method string
	var byteCode []byte
	var parameterPairs []*commonpb.KeyValuePair
	var parameters map[string]string
	var endorsements []*commonpb.EndorsementEntry
	var sequence uint64

	result := &commonpb.Result{
		Code: commonpb.TxStatusCode_SUCCESS,
		ContractResult: &commonpb.ContractResult{
			Code:    commonpb.ContractResultCode_OK,
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}

	switch tx.Header.TxType {
	case commonpb.TxType_QUERY_SYSTEM_CONTRACT, commonpb.TxType_QUERY_USER_CONTRACT:
		var payload commonpb.QueryPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = parseParameter(parameterPairs)
		} else {
			return ErrResult(result, fmt.Errorf("failed to unmarshal query payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_INVOKE_USER_CONTRACT:
		var payload commonpb.TransactPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = parseParameter(parameterPairs)
		} else {
			return ErrResult(result, fmt.Errorf("failed to unmarshal transact payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_INVOKE_SYSTEM_CONTRACT:
		var payload commonpb.SystemContractPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = parseParameter(parameterPairs)
		} else {
			return ErrResult(result, fmt.Errorf("failed to unmarshal invoke payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_UPDATE_CHAIN_CONFIG:
		var payload commonpb.SystemContractPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = parseParameter(parameterPairs)
			endorsements = payload.Endorsement
			sequence = payload.Sequence

			if endorsements == nil {
				return ErrResult(result, fmt.Errorf("endorsements not found in config update payload, tx id:%s", tx.Header.TxId))
			}
			payload.Endorsement = nil
			verifyPayloadBytes, err := proto.Marshal(&payload)

			if err = acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
				return ErrResult(result, err)
			}

			log.Debugf("chain config update [%d] [%v]", sequence, endorsements)
		} else {
			return ErrResult(result, fmt.Errorf("failed to unmarshal system contract payload for tx %s, %s", tx.Header.TxId, err.Error()))
		}
	case commonpb.TxType_MANAGE_USER_CONTRACT:
		var payload commonpb.ContractMgmtPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			if payload.ContractId == nil {
				return ErrResult(result, fmt.Errorf("param is null"))
			}
			contractName = payload.ContractId.ContractName
			runtimeType = payload.ContractId.RuntimeType
			contractVersion = payload.ContractId.ContractVersion
			method = payload.Method
			byteCode = payload.ByteCode
			parameterPairs = payload.Parameters
			parameters = parseParameter(parameterPairs)
			endorsements = payload.Endorsement

			if endorsements == nil {
				return ErrResult(result, fmt.Errorf("endorsements not found in contract mgmt payload, tx id:%s", tx.Header.TxId))
			}

			payload.Endorsement = nil
			verifyPayloadBytes, err := proto.Marshal(&payload)

			if err = acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
				return ErrResult(result, err)
			}
		} else {
			return ErrResult(result, fmt.Errorf("failed to unmarshal contract mgmt payload for tx %s, %s", tx.Header.TxId, err.Error()))
		}
	default:
		return ErrResult(result, fmt.Errorf("no such tx type: %s", tx.Header.TxType))
	}

	contractId = &commonpb.ContractId{
		ContractName:    contractName,
		ContractVersion: contractVersion,
		RuntimeType:     runtimeType,
	}

	// verify parameters
	if len(parameters) > protocol.ParametersKeyMaxCount {
		return ErrResult(result, fmt.Errorf("expect less than %d parameters, but get %d, tx id:%s", protocol.ParametersKeyMaxCount, len(parameters),
			tx.Header.TxId))
	}
	for key, val := range parameters {
		if len(key) > protocol.DefaultStateLen {
			return ErrResult(result, fmt.Errorf("expect key length less than %d, but get %d, tx id:%s", protocol.DefaultStateLen, len(key), tx.Header.TxId))
		}
		match, err := regexp.MatchString(protocol.DefaultStateRegex, key)
		if err != nil || !match {
			return ErrResult(result, fmt.Errorf("expect key no special characters, but get %s. letter, number, dot and underline are allowed, tx id:%s", key, tx.Header.TxId))
		}
		if len(val) > protocol.ParametersValueMaxLength {
			return ErrResult(result, fmt.Errorf("expect value length less than %d, but get %d, tx id:%s", protocol.ParametersValueMaxLength, len(val), tx.Header.TxId))
		}
	}
	contractResultPayload, txStatusCode := vmManager.RunContract(contractId, method, byteCode, parameters, txSimContext, 0, tx.Header.TxType)

	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	if txStatusCode == commonpb.TxStatusCode_SUCCESS {
		return result, nil
	} else {
		return result, errors.New(contractResultPayload.Message)
	}
}
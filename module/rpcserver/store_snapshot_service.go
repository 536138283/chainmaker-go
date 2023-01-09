/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v3/bytehelper"
	commonErr "chainmaker.org/chainmaker/common/v3/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
)

// dealSnapshot , deal snapshot
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) dealSnapshot(tx *commonPb.Transaction) *commonPb.TxResponse {
	if tx.Payload.TxType != commonPb.TxType_SNAPSHOT {
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
			TxId:    tx.Payload.TxId,
		}
	}

	switch tx.Payload.Method {

	//make a snapshot
	case syscontract.SnapshotFunction_SNAPSHOT_MAKE.String():
		return s.makeSnapshot(tx)
	//get status
	case syscontract.SnapshotFunction_SNAPSHOT_GET_STATUS.String():
		return s.getSnapshotStatus(tx)
	default:
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
		}
	}
}

// makeSnapshot , make a snapshot of store
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) makeSnapshot(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err         error
		errMsg      string
		blockHeight uint64
		errCode     commonErr.ErrCode
		store       protocol.BlockchainStore
		resp        = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId
	// get store from map
	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}
	// get snapshot height by tx payload parameters
	if blockHeight, err = s.getSnapshotHeight(tx.Payload.Parameters); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SNAPSHOT_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}
	// make a snapshot by using the  height
	if err = store.MakeSnapshot(blockHeight); err != nil {
		errMsg = fmt.Sprintf("make snapshot failed, %s", err.Error())
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = commonPb.TxStatusCode_SUCCESS.String()
	return resp
}

// getSnapshotHeight , get snapshot height from store
// @param []*commonPb.KeyValuePair
// @return uint64
// @return error
func (s *ApiService) getSnapshotHeight(params []*commonPb.KeyValuePair) (uint64, error) {
	if len(params) != 1 {
		return 0, errors.New("params count != 1")
	}

	key := syscontract.SnapshotHeight_SNAPSHOT_HEIGHT.String()
	if params[0].Key != key {
		return 0, fmt.Errorf("invalid key, must be %s", key)
	}

	blockHeight, err := bytehelper.BytesToUint64(params[0].Value)
	if err != nil {
		return 0, errors.New("convert blockHeight from bytes to uint64 failed")
	}

	return blockHeight, nil
}

// getSnapshotStatus , get snapshot status by tx
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) getSnapshotStatus(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err    error
		errMsg string
		//blockHeight uint64
		errCode commonErr.ErrCode
		store   protocol.BlockchainStore
		resp    = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId
	// get store from map
	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	status := store.GetSnapshotStatus()
	if status == 0 {
		resp.Code = commonPb.TxStatusCode_MAKE_SNAPSHOT_STATUS_UNFINISHED
		resp.Message = commonPb.TxStatusCode_SUCCESS.String()
		return resp
	}
	if status == 1 {
		resp.Code = commonPb.TxStatusCode_MAKE_SNAPSHOT_STATUS_FINISH
		resp.Message = commonPb.TxStatusCode_SUCCESS.String()
		return resp
	}

	errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SNAPSHOT_BLOCK
	errMsg = s.getErrMsg(errCode, err)
	s.log.Error(errMsg)
	resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
	resp.Message = errMsg
	return resp

}

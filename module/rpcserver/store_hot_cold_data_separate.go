/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v3/bytehelper"
	commonErr "chainmaker.org/chainmaker/common/v3/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	storePb "chainmaker.org/chainmaker/pb-go/v3/store"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
)

// dealHotColdDataSeparate , deal hot-cold-data-separate
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) dealHotColdDataSeparate(tx *commonPb.Transaction) *commonPb.TxResponse {
	if tx.Payload.TxType != commonPb.TxType_HOT_COLD_DATA_SEPARATION {
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
			TxId:    tx.Payload.TxId,
		}
	}

	switch tx.Payload.Method {

	//do a hot cold separate job
	case syscontract.HotColdDataSeparateFunction_DoHotColdDataSeparation.String():
		return s.doHotColdDataSeparation(tx)
	//get job status
	case syscontract.HotColdDataSeparateFunction_GetHotColdDataSeparationJobByID.String():
		return s.getHotColdDataSeparationJobByID(tx)
	default:
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
		}
	}
}

// doHotColdDataSeparation , do a hot cold separate job, then return jobID in tx.message
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) doHotColdDataSeparation(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err                    error
		errMsg, jobID          string
		startHeight, endHeight uint64
		errCode                commonErr.ErrCode
		store                  protocol.BlockchainStore
		resp                   = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId

	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if startHeight, endHeight, err = s.getStartEndHeight(tx.Payload.Parameters); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_HOT_COLD_SEPARATE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if jobID, err = store.DoHotColdDataSeparation(startHeight, endHeight); err != nil {
		errMsg = fmt.Sprintf("make hot cold data separate failed, %s", err.Error())
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = jobID
	return resp
}

// getStartEndHeight , get start height, end height from params
// @param []*commonPb.KeyValuePair
// @return uint64
// @return uint64
// @return error
func (s *ApiService) getStartEndHeight(params []*commonPb.KeyValuePair) (uint64, uint64, error) {
	if len(params) != 2 {
		return 0, 0, errors.New("params count != 2")
	}

	startHeightKey := syscontract.DoHotColdDataSeparateHeight_START_HEIGHT.String()
	if params[0].Key != startHeightKey {
		return 0, 0, fmt.Errorf("invalid key, must be %s", startHeightKey)
	}

	startHeight, err := bytehelper.BytesToUint64(params[0].Value)
	if err != nil {
		return 0, 0, errors.New("convert blockHeight from bytes to uint64 failed")
	}

	endHeightKey := syscontract.DoHotColdDataSeparateHeight_END_HEIGHT.String()
	if params[1].Key != endHeightKey {
		return 0, 0, fmt.Errorf("invalid key, must be %s", endHeightKey)
	}

	endHeight, err := bytehelper.BytesToUint64(params[1].Value)
	if err != nil {
		return 0, 0, errors.New("convert blockHeight from bytes to uint64 failed")
	}

	return startHeight, endHeight, nil
}

// getHotColdDataSeparationJobByID, get hot cold separate job by jobID, return jobInfo json in resp message
// @param *commonPb.Transaction
// @return *commonPb.TxResponse
func (s *ApiService) getHotColdDataSeparationJobByID(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err           error
		errMsg, jobID string
		//blockHeight uint64
		errCode     commonErr.ErrCode
		store       protocol.BlockchainStore
		resp        = &commonPb.TxResponse{TxId: tx.Payload.TxId}
		job         storePb.ArchiveJob
		respMessage []byte
	)

	chainId := tx.Payload.ChainId

	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if jobID, err = s.getJobID(tx.Payload.Parameters); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if job, err = store.GetHotColdDataSeparationJobByID(jobID); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	//jobInfo to json
	if respMessage, err = json.Marshal(job); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}
	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = string(respMessage)
	return resp

}

// getStartEndHeight , get start height, end height from params
// @param []*commonPb.KeyValuePair
// @return string
// @return error
func (s *ApiService) getJobID(params []*commonPb.KeyValuePair) (string, error) {
	if len(params) != 1 {
		return "", errors.New("params count != 1")
	}

	key := syscontract.GetHotColdDataSeparateJob_JOB_ID.String()
	if params[0].Key != key {
		return "", fmt.Errorf("invalid key, must be %s", key)
	}

	jobID := string(params[0].Value)

	return jobID, nil
}

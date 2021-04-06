package common

import commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"

func ErrResult(result *commonpb.Result, err error) (*commonpb.Result, error) {
	result.ContractResult.Message = err.Error()
	result.Code = commonpb.TxStatusCode_INVALID_PARAMETER
	result.ContractResult.Code = commonpb.ContractResultCode_FAIL
	return result, err
}
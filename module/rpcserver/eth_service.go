package rpcserver

import (
	"fmt"

	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

// dealEthTx - deal ethereum tx
func (s *ApiService) dealEthTx(tx *commonPb.Transaction, source protocol.TxSource) *commonPb.TxResponse {
	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		resp    = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	err = s.chainMakerServer.AddTx(tx.Payload.ChainId, tx, source)

	s.incInvokeCounter(tx.Payload.ChainId, err)
	s.updateTxSizeHistogram(tx, err)

	if err != nil {
		errMsg = fmt.Sprintf("Add tx failed, %s, chainId:%s, txId:%s",
			err.Error(), tx.Payload.ChainId, tx.Payload.TxId)
		s.log.Warn(errMsg)

		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		resp.TxId = tx.Payload.TxId
		return resp
	}

	s.log.Debugf("Add tx success, chainId:%s, txId:%s", tx.Payload.ChainId, tx.Payload.TxId)

	errCode = commonErr.ERR_CODE_OK
	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = errCode.String()
	resp.TxId = tx.Payload.TxId
	return resp
}

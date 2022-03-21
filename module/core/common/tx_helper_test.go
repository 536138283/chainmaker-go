package common

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

func TestValidateTx(t *testing.T) {
	verifyTx, block := txPrepare(t)
	hashes, _, _, err := verifyTx.verifierTxs(block)
	require.Nil(t, err)

	for _, hash := range hashes {
		fmt.Println("test hash: ", hex.EncodeToString(hash))
	}
}

func newTx(txId string, contractId *commonpb.Contract, parameterMap map[string]string) *commonpb.Transaction {

	var parameters []*commonpb.KeyValuePair
	for key, value := range parameterMap {
		parameters = append(parameters, &commonpb.KeyValuePair{
			Key:   key,
			Value: []byte(value),
		})
	}

	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "chain1",
			TxType:         commonpb.TxType_QUERY_CONTRACT,
			TxId:           txId,
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   contractId.Name,
			Method:         "set",
			Parameters:     parameters,
			Sequence:       0,
			Limit:          nil,
		},
		Sender:    nil,
		Endorsers: nil,
		Result: &commonpb.Result{
			Code: 0,
			ContractResult: &commonpb.ContractResult{
				Code:          0,
				Result:        nil,
				Message:       "",
				GasUsed:       0,
				ContractEvent: nil,
			},
			RwSetHash: nil,
		},
	}

}

func txPrepare(t *testing.T) (*VerifierTx, *commonpb.Block) {
	block := newBlock()
	//contractId := &commonpb.Contract{
	//	ContractName:    "ContractName",
	//	ContractVersion: "1",
	//	RuntimeType:     commonpb.RuntimeType_WASMER,
	//}
	contractId := &commonpb.Contract{
		Name:        "ContractName",
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_DOCKER_GO,
		Status:      0,
		Creator:     nil,
	}

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	txs := make([]*commonpb.Transaction, 0)
	txs = append(txs, tx0)
	block.Txs = txs

	var txRWSetMap = make(map[string]*commonpb.TxRWSet, 3)
	txRWSetMap[tx0.Payload.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}

	rwHash, _ := hex.DecodeString("d02f421ed76e0e26e9def824a8b84c7c223d484762d6d060a8b71e1649d1abbf")
	result := &commonpb.Result{
		Code: commonpb.TxStatusCode_SUCCESS,
		ContractResult: &commonpb.ContractResult{
			Code:    0,
			Result:  nil,
			Message: "",
			GasUsed: 0,
		},
		RwSetHash: rwHash,
	}
	tx0.Result = result
	txResultMap := make(map[string]*commonpb.Result, 1)
	txResultMap[tx0.Payload.TxId] = result

	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")

	ctl := gomock.NewController(t)
	store := mock.NewMockBlockchainStore(ctl)
	txPool := mock.NewMockTxPool(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	chainConf := mock.NewMockChainConf(ctl)

	store.EXPECT().TxExists(tx0).AnyTimes().Return(false, nil)

	txsMap := make(map[string]*commonpb.Transaction)

	txsMap[tx0.Payload.TxId] = tx0

	txPool.EXPECT().GetTxsByTxIds([]string{tx0.Payload.TxId}).Return(txsMap, nil)
	//config := &config.ChainConfig{
	//	ChainId: "chain1",
	//	Crypto: &config.CryptoConfig{
	//		Hash: "SHA256",
	//	},
	//}
	config := &config.ChainConfig{
		ChainId:   "chain1",
		Version:   "1.0",
		Crypto:    &config.CryptoConfig{Hash: "SHA256"},
		Consensus: &config.ConsensusConfig{Type: 0},
		Core:      &config.CoreConfig{},
	}

	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)

	principal := mock.NewMockPrincipal(ctl)
	ac.EXPECT().CreatePrincipal("123", nil, nil).AnyTimes().Return(principal, nil)
	ac.EXPECT().VerifyPrincipal(principal).AnyTimes().Return(true, nil)
	verifyTxConf := &VerifierTxConfig{
		Block:       block,
		TxRWSetMap:  txRWSetMap,
		TxResultMap: txResultMap,
		Store:       store,
		TxPool:      txPool,
		Ac:          ac,
		ChainConf:   chainConf,
		Log:         log,
	}
	return NewVerifierTx(verifyTxConf), block
}

func newBlock() *commonpb.Block {
	var hash = []byte("0123456789")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockVersion:   1,
			BlockType:      0,
			ChainId:        "chain1",
			BlockHeight:    3,
			BlockHash:      hash,
			PreBlockHash:   hash,
			PreConfHeight:  0,
			TxCount:        0,
			TxRoot:         hash,
			DagHash:        hash,
			RwSetRoot:      hash,
			BlockTimestamp: 0,
			ConsensusArgs:  hash,
			Proposer: &accesscontrol.Member{
				OrgId:      "org1",
				MemberType: 0,
				MemberInfo: hash,
			},
			Signature: hash,
		},
		Dag:            &commonpb.DAG{Vertexes: nil},
		Txs:            nil,
		AdditionalData: nil,
	}

	tx := createNewTestTx("txId1")
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}

func TestIfExitInSameBranch(t *testing.T) {

	tx1 := createNewTestTx("123456")
	tx2 := createNewTestTx("1234567")
	tx3 := createNewTestTx("1234568")
	tx4 := createNewTestTx("1234569")
	tx5 := createNewTestTx("12345610")

	b0 := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  9,
			BlockHash:    []byte("012345"),
			PreBlockHash: []byte("012345"),
		},
		Txs: nil,
	}

	b1 := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  10,
			BlockHash:    []byte("0123456"),
			PreBlockHash: []byte("012345"),
		},
		Txs: []*commonpb.Transaction{tx1, tx2},
	}

	b2 := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  11,
			BlockHash:    []byte("123"),
			PreBlockHash: []byte("0123456"),
		},
		Txs: []*commonpb.Transaction{tx3},
	}

	b2a := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  11,
			BlockHash:    []byte("123a"),
			PreBlockHash: []byte("0123456"),
		},
		Txs: []*commonpb.Transaction{tx2},
	}

	b3 := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  12,
			BlockHash:    []byte("1234"),
			PreBlockHash: []byte("123"),
		},
		Txs: []*commonpb.Transaction{tx4},
	}

	b3a := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  12,
			BlockHash:    []byte("1234a"),
			PreBlockHash: []byte("123"),
		},
		Txs: []*commonpb.Transaction{tx2},
	}

	b3b := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  12,
			BlockHash:    []byte("1234b"),
			PreBlockHash: []byte("123"),
		},
		Txs: []*commonpb.Transaction{tx5, tx3},
	}

	b4 := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  13,
			BlockHash:    []byte("12345"),
			PreBlockHash: []byte("1234"),
		},
		Txs: []*commonpb.Transaction{tx1},
	}

	b4a := commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:  13,
			BlockHash:    []byte("12345"),
			PreBlockHash: []byte("1234"),
		},
		Txs: []*commonpb.Transaction{tx5},
	}

	ctl := gomock.NewController(t)
	proposalCache := mock.NewMockProposalCache(ctl)
	proposalCache.EXPECT().GetProposedBlockByHashAndHeight(b0.Header.BlockHash, b0.Header.BlockHeight).Return(nil, nil).AnyTimes()
	cases := []struct {
		b0       *commonpb.Block
		b1       *commonpb.Block
		preBlock *commonpb.Block
		block    *commonpb.Block
		doc      string
		expected bool // expected result
	}{
		/**
								-> b3a
		 						-> b3b

						 -> b2

								-> b3  ->   b4
									   ->	b4a
				b0 -> b1
						 -> b2a

		*/
		{nil, nil, &b1, &b2, "区块b2里的交易与前面的区块的交易不重复", false},
		{nil, nil, &b1, &b2a, "区块b2a里的交易与b1的区块的交易重复", true},
		{nil, &b1, &b2, &b3a, "区块b3a里的交易与b1的区块的交易重复", true},
		{nil, &b1, &b2, &b3b, "区块b3b里的交易与b2的区块的交易重复", true},
		{nil, &b1, &b2, &b3, "区块b3里的交易与前面的区块的交易不重复", false},
		//
		{&b1, &b2, &b3, &b4, "区块b4里的交易与b1的区块的交易重复", true},
		{&b1, &b2, &b3, &b4a, "区块b4a里的交易与b3b的区块的交易重复", false},
	}

	for i, v := range cases {
		proposalCachePrepare(proposalCache, v.b0, v.b1, v.preBlock)

		var finalResult bool
		for _, tx := range v.block.Txs {
			result := ifExitInSameBranch(
				v.block.Header.BlockHeight,
				tx.Payload.TxId,
				proposalCache,
				v.block.Header.PreBlockHash)

			if result {
				finalResult = true
			}
		}

		if finalResult != v.expected {
			fmt.Printf("Case:%d fail \n", i)
			require.Equal(t, v.expected, finalResult)
		} else {
			fmt.Printf("Case:%d pass \n", i)
		}

	}

}

func proposalCachePrepare(proposalCache *mock.MockProposalCache, b0, b1, preBlock *commonpb.Block) {
	proposalCache.EXPECT().GetProposedBlockByHashAndHeight(
		preBlock.Header.BlockHash,
		preBlock.Header.BlockHeight).
		Return(preBlock, nil).AnyTimes()

	if b0 != nil {
		proposalCache.EXPECT().GetProposedBlockByHashAndHeight(
			b0.Header.BlockHash,
			b0.Header.BlockHeight).
			Return(b0, nil).AnyTimes()
	}

	if b1 != nil {
		proposalCache.EXPECT().GetProposedBlockByHashAndHeight(
			b1.Header.BlockHash,
			b1.Header.BlockHeight).
			Return(b1, nil).AnyTimes()
	}

}

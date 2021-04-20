package common

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestFinalizeBlock(t *testing.T) {
	ctl := gomock.NewController(t)
	identity := mock.NewMockSigningMember(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	identity.EXPECT().Serialize(true).AnyTimes().Return([]byte("DEFAULTPROPOSER"),nil)
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(nil)

	block := cache.CreateNewTestBlock(0)
	chainId := "123"
	nblock, er := InitNewBlock(block,identity,chainId,chainConf)
	fmt.Println(er)

	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	aclFailTxs := make([]*commonpb.Transaction,0,0)
	hashtype := "456"
	er = FinalizeBlock(nblock,txRWSetMap,aclFailTxs,hashtype)
	fmt.Println(er)
}
package abft

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker-go/core/cache"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

var (
	contractName = "testContract"
)

func TestMerger_Merge(t *testing.T) {

	branchID1 := []byte("a")
	branchID2 := []byte("b")
	branchID3 := []byte("c")
	branchID4 := []byte("d")

	//cach := addTxBatch_NoRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4)
	//cach := addTxBatch_NoRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4)
	//cach := addTxBatch_HasRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4)
	//cach := addTxBatch_HasRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4)
	cach := addTxBatch_HasRepeatTx_HasConflic_2(branchID1, branchID2, branchID3, branchID4)

	m := NewMerger()
	c := &Committer{
		merger:        m,
		retryList:     nil,
		abftCache:     cach,
		txBatchIDList: make([]string, 0),
	}

	txBatchHash := [][]byte{branchID3, branchID2, branchID4, branchID1}
	c.prepare(txBatchHash)
	c.sortTxBatchID()
	fmt.Println(c.txBatchIDList)

	block := cache.CreateNewTestBlock(3)

	if err := c.merger.Merge(block, c.txBatchIDList); err != nil {
		panic(err)
	}

	fmt.Println("rwSetMap:", c.merger.rwSetMap)
	fmt.Println("block.dag:", block.Dag)
	fmt.Println("Txs num:", len(block.Txs))
	fmt.Println("Txs:", block.Txs)

}

func getTxsForMerge() []*commonpb.Transaction {
	contractId := &commonpb.ContractId{
		ContractName:    contractName,
		ContractVersion: "1",
		RuntimeType:     commonpb.RuntimeType_WASMER,
	}
	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	tx1 := newTx("a0000000000000000000000000000001", contractId, parameters)
	tx2 := newTx("a0000000000000000000000000000002", contractId, parameters)
	tx3 := newTx("a0000000000000000000000000000003", contractId, parameters)
	tx4 := newTx("a0000000000000000000000000000004", contractId, parameters)
	tx5 := newTx("a0000000000000000000000000000005", contractId, parameters)
	tx6 := newTx("a0000000000000000000000000000006", contractId, parameters)
	tx7 := newTx("a0000000000000000000000000000007", contractId, parameters)
	tx8 := newTx("a0000000000000000000000000000008", contractId, parameters)
	tx9 := newTx("a0000000000000000000000000000009", contractId, parameters)
	tx10 := newTx("a0000000000000000000000000000010", contractId, parameters)
	tx11 := newTx("a00000000000000000000000000000011", contractId, parameters)
	tx12 := newTx("a00000000000000000000000000000012", contractId, parameters)

	txList := []*commonpb.Transaction{tx0, tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12}

	return txList
}

func addTxBatch_NoRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4 []byte) *cache.AbftCache {
	txList := getTxs()
	tx0 := txList[0]
	tx1 := txList[1]
	tx2 := txList[2]
	tx3 := txList[3]
	tx4 := txList[4]
	tx5 := txList[5]
	tx6 := txList[6]
	tx7 := txList[7]

	hc := cache.NewAbftCache()
	m := NewMerger()
	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
	rwSetMap0[tx0.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	hash0 := branchID1
	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1})
	b0.Dag = m.buildDAG(b0, rwSetMap0)
	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
	//hc.SetProposedTxBatch(b0, rwSetMap0)

	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
	rwSetMap1[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	hash1 := branchID2
	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx3})
	b1.Dag = m.buildDAG(b1, rwSetMap1)
	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
	//hc.SetProposedTxBatch(b1, rwSetMap1)

	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
	rwSetMap2[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K10"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}

	hash2 := branchID3
	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx5})
	b2.Dag = m.buildDAG(b2, rwSetMap2)
	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
	//hc.SetProposedTxBatch(b2, rwSetMap2)

	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
	rwSetMap3[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	hash3 := branchID4
	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7})
	b3.Dag = m.buildDAG(b3, rwSetMap3)
	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
	//hc.SetProposedTxBatch(b3, rwSetMap3)

	return hc
}

func addTxBatch_NoRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4 []byte) *cache.AbftCache {
	txList := getTxsForMerge()
	tx0 := txList[0]
	tx1 := txList[1]
	tx2 := txList[2]
	tx3 := txList[3]
	tx4 := txList[4]
	tx5 := txList[5]
	tx6 := txList[6]
	tx7 := txList[7]
	tx8 := txList[8]
	tx9 := txList[9]
	tx10 := txList[10]
	tx11 := txList[11]

	hc := cache.NewAbftCache()
	m := NewMerger()
	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
	rwSetMap0[tx0.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx8.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx8.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	hash0 := branchID1
	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx8})
	b0.Dag = m.buildDAG(b0, rwSetMap0)
	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
	//hc.SetProposedTxBatch(b0, rwSetMap0)

	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
	rwSetMap1[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx9.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx9.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
	}
	hash1 := branchID2
	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx3, tx9})
	b1.Dag = m.buildDAG(b1, rwSetMap1)
	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
	//hc.SetProposedTxBatch(b1, rwSetMap1)

	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
	rwSetMap2[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx10.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx10.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K10"),
			Value:        []byte("V"),
		}},
	}

	hash2 := branchID3
	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx5, tx10})
	b2.Dag = m.buildDAG(b2, rwSetMap2)
	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
	//hc.SetProposedTxBatch(b2, rwSetMap2)

	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
	rwSetMap3[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx11.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx11.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	hash3 := branchID4
	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7, tx11})
	b3.Dag = m.buildDAG(b3, rwSetMap3)
	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
	//hc.SetProposedTxBatch(b3, rwSetMap3)

	return hc
}

func addTxBatch_HasRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4 []byte) *cache.AbftCache {
	txList := getTxs()
	tx0 := txList[0]
	tx1 := txList[1]
	tx2 := txList[2]
	tx3 := txList[3]
	tx4 := txList[4]
	tx5 := txList[5]
	tx6 := txList[6]
	tx7 := txList[7]

	hc := cache.NewAbftCache()
	m := NewMerger()
	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
	rwSetMap0[tx0.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	hash0 := branchID1
	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1})
	b0.Dag = m.buildDAG(b0, rwSetMap0)
	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
	//hc.SetProposedTxBatch(b0, rwSetMap0)

	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
	rwSetMap1[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	hash1 := branchID2
	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx1, tx2, tx3})
	b1.Dag = m.buildDAG(b1, rwSetMap1)
	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
	//hc.SetProposedTxBatch(b1, rwSetMap1)

	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
	rwSetMap2[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K10"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}

	hash2 := branchID3
	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx3, tx5})
	b2.Dag = m.buildDAG(b2, rwSetMap2)
	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
	//hc.SetProposedTxBatch(b2, rwSetMap2)

	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
	rwSetMap3[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K15"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K16"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	hash3 := branchID4
	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7, tx5, tx3})
	b3.Dag = m.buildDAG(b3, rwSetMap3)
	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
	//hc.SetProposedTxBatch(b3, rwSetMap3)

	return hc
}

func addTxBatch_HasRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4 []byte) *cache.AbftCache {
	txList := getTxsForMerge()
	tx0 := txList[0]
	tx1 := txList[1]
	tx2 := txList[2]
	tx3 := txList[3]
	tx4 := txList[4]
	tx5 := txList[5]
	tx6 := txList[6]
	tx7 := txList[7]
	tx8 := txList[8]
	tx9 := txList[9]
	tx10 := txList[10]
	tx11 := txList[11]

	hc := cache.NewAbftCache()
	m := NewMerger()
	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
	rwSetMap0[tx0.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K6"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	hash0 := branchID1
	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx2, tx3})
	b0.Dag = m.buildDAG(b0, rwSetMap0)
	fmt.Println(b0.Dag)
	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
	//hc.SetProposedTxBatch(b0, rwSetMap0)

	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
	rwSetMap1[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K10"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	hash1 := branchID2
	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx3, tx4, tx5, tx6})
	b1.Dag = m.buildDAG(b1, rwSetMap1)
	fmt.Println(b1.Dag)
	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
	//hc.SetProposedTxBatch(b1, rwSetMap1)

	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
	rwSetMap2[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx8.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx8.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K15"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx9.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx9.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K16"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K17"),
			Value:        []byte("V"),
		}},
	}

	hash2 := branchID3
	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx7, tx3, tx8, tx9})
	b2.Dag = m.buildDAG(b2, rwSetMap2)
	fmt.Println(b2.Dag)
	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
	//hc.SetProposedTxBatch(b2, rwSetMap2)

	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
	rwSetMap3[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx10.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx10.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K18"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx11.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx11.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K19"),
			Value:        []byte("V"),
		}},
	}
	hash3 := branchID4
	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx7, tx10, tx6, tx11})
	b3.Dag = m.buildDAG(b3, rwSetMap3)
	fmt.Println(b3.Dag)
	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
	//hc.SetProposedTxBatch(b3, rwSetMap3)

	return hc
}

func addTxBatch_HasRepeatTx_HasConflic_2(branchID1, branchID2, branchID3, branchID4 []byte) *cache.AbftCache {
	txList := getTxsForMerge()
	tx0 := txList[0]
	tx1 := txList[1]
	tx2 := txList[2]
	tx3 := txList[3]
	tx4 := txList[4]
	tx5 := txList[5]
	tx6 := txList[6]
	tx7 := txList[7]
	tx8 := txList[8]
	tx9 := txList[9]
	tx10 := txList[10]
	tx11 := txList[11]
	tx12 := txList[12]

	hc := cache.NewAbftCache()
	m := NewMerger()
	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
	rwSetMap0[tx0.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx1.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap0[tx3.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx3.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K7"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	hash0 := branchID1
	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx2, tx3})
	b0.Dag = m.buildDAG(b0, rwSetMap0)
	fmt.Println(b0.Dag)
	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
	//hc.SetProposedTxBatch(b0, rwSetMap0)

	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
	rwSetMap1[tx2.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx2.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx5.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx5.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K9"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap1[tx6.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx6.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K11"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K12"),
			Value:        []byte("V"),
		}},
	}
	hash1 := branchID2
	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx4, tx5, tx6})
	b1.Dag = m.buildDAG(b1, rwSetMap1)
	fmt.Println(b1.Dag)
	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
	//hc.SetProposedTxBatch(b1, rwSetMap1)

	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
	rwSetMap2[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx4.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx4.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K5"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx8.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx8.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K8"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K15"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap2[tx9.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx9.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K16"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K17"),
			Value:        []byte("V"),
		}},
	}

	hash2 := branchID3
	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx7, tx4, tx8, tx9})
	b2.Dag = m.buildDAG(b2, rwSetMap2)
	fmt.Println(b2.Dag)
	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
	//hc.SetProposedTxBatch(b2, rwSetMap2)

	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
	rwSetMap3[tx7.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx7.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K13"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx10.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx10.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K14"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K18"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx11.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx11.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K4"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K19"),
			Value:        []byte("V"),
		}},
	}
	rwSetMap3[tx12.Header.TxId] = &commonpb.TxRWSet{
		TxId: tx12.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K19"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K20"),
			Value:        []byte("V"),
		}},
	}
	hash3 := branchID4
	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx7, tx10, tx11, tx12})
	b3.Dag = m.buildDAG(b3, rwSetMap3)
	fmt.Println(b3.Dag)
	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
	//hc.SetProposedTxBatch(b3, rwSetMap3)

	return hc
}

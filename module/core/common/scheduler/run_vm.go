package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
	"golang.org/x/sync/singleflight"

	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
)

var sf singleflight.Group

// guardForExecuteTx2220
// filter out txs that need not go into runVM(...)
// returns
// 		willExit: bool
func (ts *TxScheduler) guardForExecuteTx2220(tx *commonPb.Transaction, txSimContext protocol.TxSimContext,
	enableGas bool, enableOptimizedChargeGas bool) (
	txIsAllow bool) {
	if tx.Result != nil && tx.Result.Code == commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED {
		if enableOptimizedChargeGas {
			txSimContext.SetTxResult(tx.Result)
			return false
		}
	}

	return true
}

// guardForExecuteTx300 guard for execute tx after 300 version
func (ts *TxScheduler) guardForExecuteTx300(
	tx *commonPb.Transaction, txSimContext protocol.TxSimContext, enableGas bool,
	enableOptimizeChargeGas bool, snapshot protocol.Snapshot, collection *SenderCollection) (txIsAllow bool) {
	txNeedChargeGas := ts.checkNativeFilter(
		tx.Payload.ContractName,
		tx.Payload.Method,
		tx,
		txSimContext.GetSnapshot())

	switch {
	case enableOptimizeChargeGas:
		return ts.guardForExecuteTx300WithOptimizeChargeGas(tx, txSimContext, txNeedChargeGas, snapshot, collection)
	case enableGas:
		return ts.guardForExecuteTx300WithChargeGas(tx, txSimContext, txNeedChargeGas)
	default:
		return true
	}
}

// guardForExecuteTx300WithChargeGas guard for execute tx with charge gas after 300 version
func (ts *TxScheduler) guardForExecuteTx300WithChargeGas(
	tx *commonPb.Transaction, txSimContext protocol.TxSimContext, txNeedChargeGas bool) (txIsAllow bool) {

	if !txNeedChargeGas {
		return true
	}

	if tx.Payload.Limit == nil {
		setTxResultForGasLimitNotSet(txSimContext)
		if tx.Result == nil {
			tx.Result = txSimContext.GetTxResult()
		}
		return false
	}

	return true
}

// guardForExecuteTx300WithOptimizeChargeGas guard for execute tx with optimize charge gas after 300 version
func (ts *TxScheduler) guardForExecuteTx300WithOptimizeChargeGas(
	tx *commonPb.Transaction, txSimContext protocol.TxSimContext, txNeedChargeGas bool,
	snapshot protocol.Snapshot, senderCollection *SenderCollection) (txIsAllow bool) {

	// 不需要gas扣费
	if !txNeedChargeGas {
		return true
	}

	// 未设置gaslimit，直接报错
	if tx.Payload.Limit == nil {
		setTxResultForGasLimitNotSet(txSimContext)
		return false
	}

	// 开启了gas优化，但是senderCollection为空，则执行交易
	if senderCollection == nil || senderCollection.txsMap == nil {
		return true
	}

	ts.log.DebugDynamic(func() string {
		return fmt.Sprintf("begin check [txid:%s]account balance", tx.Payload.TxId)
	})

	chainCfg := snapshot.GetLastChainConfig()

	// get the public key from tx
	pk, err := getPkFromTx(tx, snapshot)
	if err != nil {
		ts.log.Errorf("getPkFromTx failed: err = %v", err)
		return false
	}

	// convert the public key to `ZX` or `CM` or `EVM` address
	addr, err := publicKeyToAddress(pk, chainCfg)
	if err != nil {
		ts.log.Error("publicKeyToAddress failed: err = %v", err)
		return false
	}

	value, _ := senderCollection.txsMap.Load(addr)
	collection, ok := value.(*TxCollection)
	if !ok {
		ts.log.Error("get tx collection fail")
		return false
	}

	collection.mu.Lock()
	defer collection.mu.Unlock()

	balance := collection.accountBalance
	limit := int64(tx.Payload.Limit.GasLimit)

	// 校验余额是否足够
	if balance-limit < 0 {
		pkStr, _ := collection.publicKey.String()
		ts.log.Debugf("balance is too low to execute tx. address = %v, public key = %s", addr, pkStr)
		setTxResultForGasBalanceNotEnough(txSimContext, addr)

		if tx.Result == nil {
			tx.Result = txSimContext.GetTxResult()
		}

		return false
	}

	collection.accountBalance = collection.accountBalance - limit
	senderCollection.txsMap.Store(addr, collection)

	return true
}

// setTxResultForGasBalanceNotEnough set tx result about gas balance not enough
func setTxResultForGasBalanceNotEnough(txSimContext protocol.TxSimContext, addr string) {
	errMsg := fmt.Sprintf("`%s` has no enough balance to execute tx.", addr)
	txResult := &commonPb.Result{
		Code: commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED,
		ContractResult: &commonPb.ContractResult{
			Code:    uint32(1),
			Result:  nil,
			Message: errMsg,
			GasUsed: uint64(0),
		},
		RwSetHash: nil,
		Message:   errMsg,
	}

	txSimContext.SetTxResult(txResult)
}

// setTxResultForGasLimitNotSet set tx result about gas limit not set
func setTxResultForGasLimitNotSet(txSimContext protocol.TxSimContext) {
	txResult := &commonPb.Result{
		Code: commonPb.TxStatusCode_GAS_LIMIT_NOT_SET,
		ContractResult: &commonPb.ContractResult{
			Code:    uint32(1),
			Result:  nil,
			Message: ErrMsgOfGasLimitNotSet,
			GasUsed: uint64(0),
		},
		RwSetHash: nil,
		Message:   ErrMsgOfGasLimitNotSet,
	}
	txSimContext.SetTxResult(txResult)
}

func (ts *TxScheduler) guardForExecuteTx2300(tx *commonPb.Transaction, txSimContext protocol.TxSimContext,
	enableGas bool, enableOptimizeChargeGas bool, snapshot protocol.Snapshot) (txIsAllow bool) {

	txNeedChargeGas := ts.checkNativeFilter(
		tx.Payload.ContractName,
		tx.Payload.Method,
		tx,
		txSimContext.GetSnapshot())

	if enableOptimizeChargeGas {
		// below code is in charge_gas_optimize mode
		// need charge gas, but gasLimit is not set
		if txNeedChargeGas && tx.Payload.Limit == nil {
			// `verify node` should return error result same with `proposer node` do in `dispatchTxsInSenderCollection`
			txResult := &commonPb.Result{
				Code: commonPb.TxStatusCode_GAS_LIMIT_NOT_SET,
				ContractResult: &commonPb.ContractResult{
					Code:    uint32(1),
					Result:  nil,
					Message: ErrMsgOfGasLimitNotSet,
					GasUsed: uint64(0),
				},
				RwSetHash: nil,
				Message:   ErrMsgOfGasLimitNotSet,
			}
			txSimContext.SetTxResult(txResult)

			return false
		} else if txNeedChargeGas && tx.Payload.Limit != nil {
			// in `proposer node`:
			// 	1) tx.Result should be set by `dispatchTxsInSenderCollection()`
			//  2) tx.Result should be set by `runVM()`
			// in `verify node`:
			//  1) tx.Result should be set in this place
			//  2) tx.Result should be set in `runVM()` later
			if tx.Result != nil && tx.Result.Code == commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED {
				pk, _ := getPkFromTx(tx, snapshot)
				chainCfg := snapshot.GetLastChainConfig()
				addr, _ := publicKeyToAddress(pk, chainCfg)

				ts.log.DebugDynamic(func() string {
					return fmt.Sprintf("balance is too low to execute tx. address = %v, public key = %s", addr, pk)
				})

				errMsg := fmt.Sprintf("`%s` has no enough balance to execute tx.", addr)
				txResult := &commonPb.Result{
					Code: commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED,
					ContractResult: &commonPb.ContractResult{
						Code:    uint32(1),
						Result:  nil,
						Message: errMsg,
						GasUsed: uint64(0),
					},
					RwSetHash: nil,
					Message:   errMsg,
				}
				txSimContext.SetTxResult(txResult)
				return false
			}
		}
	} else if enableGas {
		// below code is in charge_gas mode

		if txNeedChargeGas && tx.Payload.Limit == nil {
			txResult := &commonPb.Result{
				Code: commonPb.TxStatusCode_GAS_LIMIT_NOT_SET,
				ContractResult: &commonPb.ContractResult{
					Code:    uint32(1),
					Result:  nil,
					Message: ErrMsgOfGasLimitNotSet,
					GasUsed: uint64(0),
				},
				RwSetHash: nil,
				Message:   ErrMsgOfGasLimitNotSet,
			}
			// `proposer node` need set result into tx and txSimContext
			// `verify node` need set result into txSimContext
			txSimContext.SetTxResult(txResult)
			if tx.Result == nil {
				tx.Result = txResult
			}
			return false
		}
	}

	return true
}

func (ts *TxScheduler) runVM2300(tx *commonPb.Transaction,
	txSimContext protocol.TxSimContext,
	enableOptimizeChargeGas bool) (
	*commonPb.Result, protocol.ExecOrderTxType, error) {
	var (
		contractName          string
		method                string
		byteCode              []byte
		pk                    []byte
		specialTxType         protocol.ExecOrderTxType
		accountMangerContract *commonPb.Contract
		contractResultPayload *commonPb.ContractResult
		txStatusCode          commonPb.TxStatusCode
		contract              *commonPb.Contract
	)

	ts.log.Debugf("runVM =>  for tx `%v`", tx.GetPayload().TxId)
	result := &commonPb.Result{
		Code: commonPb.TxStatusCode_SUCCESS,
		ContractResult: &commonPb.ContractResult{
			Code:    uint32(0),
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}
	payload := tx.Payload
	//不是查询，也不是上链的交易，则不需要VM运行
	if payload.TxType != commonPb.TxType_QUERY_CONTRACT &&
		!payload.TxType.IsBlockTx() {
		return errResult(result, fmt.Errorf("no such tx type: %s", tx.Payload.TxType))
	}

	contractName = payload.ContractName
	method = payload.Method
	parameters, err := ts.parseParameter2220(payload.Parameters, !enableOptimizeChargeGas)
	if err != nil {
		ts.log.Errorf("parse contract[%s] parameters error:%s", contractName, err)
		return errResult(result, fmt.Errorf(
			"parse tx[%s] contract[%s] parameters error:%s",
			payload.TxId,
			contractName,
			err.Error()),
		)
	}
	//Ethereum tx, use syscontract to process
	if tx.Payload.TxType.IsEthTxType() {
		contractName = syscontract.SystemContract_ETHEREUM.String()
		method = syscontract.EthereumFunction_Unpack.String()
	}
	ts.log.Debugf("runVM => txSimContext.GetContractByName(`%s`) for tx `%v`", contractName, tx.GetPayload().TxId)

	if contract, err = ts.getContractFromCache(txSimContext, contractName); err != nil {
		//ct, err, _ := sf.Do(contractName, func() (interface{}, error) {
		//	return txSimContext.GetContractByName(contractName)
		//})
		//if err != nil {
		ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
		return errResult(result, err)
	}
	//contract, ok := ct.(*commonPb.Contract)
	//if !ok {
	//	err = errors.New("failed to transfer contract from interface to struct")
	//	ts.log.Error(err)
	//	return errResult(result, err)
	//}

	if byteCode, err = ts.getContractBytecode(txSimContext, contract); err != nil {
		return errResult(result, err)
	}

	if ts.checkGasEnable() && !enableOptimizeChargeGas {
		accountMangerContract, pk, err = ts.getAccountMgrContractAndPk(txSimContext, tx, contract.Name, method)
		if err != nil {
			return result, specialTxType, err
		}

		_, err = ts.chargeGasLimit(accountMangerContract, tx, txSimContext, contract.Name, method, pk, result)
		if err != nil {
			ts.log.Errorf("charge gas limit err is %v", err)
			result.Code = commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED
			result.Message = err.Error()
			result.ContractResult.Code = uint32(1)
			result.ContractResult.Message = err.Error()
			return result, specialTxType, err
		}
	}

	contractResultPayload, specialTxType, txStatusCode = ts.VmManager.RunContract(contract, method, byteCode,
		parameters, txSimContext, 0, tx.Payload.TxType)

	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	// refund gas
	if ts.checkGasEnable() {
		// check if this invoke needs charging gas
		if !ts.checkNativeFilter(contract.Name, method, tx, txSimContext.GetSnapshot()) {
			return result, specialTxType, err
		}

		// check and refund gas
		if err = ts.checkRefundGas(accountMangerContract, tx, txSimContext, contractName, method, pk, result,
			contractResultPayload, enableOptimizeChargeGas); err != nil {
			return result, specialTxType, err
		}
	}

	if txStatusCode == commonPb.TxStatusCode_SUCCESS {
		return result, specialTxType, nil
	}
	return result, specialTxType, errors.New(contractResultPayload.Message)
}

func (ts *TxScheduler) runVM2220(tx *commonPb.Transaction,
	txSimContext protocol.TxSimContext,
	enableOptimizeChargeGas bool) (
	*commonPb.Result, protocol.ExecOrderTxType, error) {
	var (
		contractName          string
		method                string
		byteCode              []byte
		pk                    []byte
		specialTxType         protocol.ExecOrderTxType
		accountMangerContract *commonPb.Contract
		contractResultPayload *commonPb.ContractResult
		txStatusCode          commonPb.TxStatusCode
	)

	ts.log.Debugf("runVM =>  for tx `%v`", tx.GetPayload().TxId)
	result := &commonPb.Result{
		Code: commonPb.TxStatusCode_SUCCESS,
		ContractResult: &commonPb.ContractResult{
			Code:    uint32(0),
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}
	payload := tx.Payload
	if payload.TxType != commonPb.TxType_QUERY_CONTRACT && payload.TxType != commonPb.TxType_INVOKE_CONTRACT {
		return errResult(result, fmt.Errorf("no such tx type: %s", tx.Payload.TxType))
	}

	contractName = payload.ContractName
	method = payload.Method
	parameters, err := ts.parseParameter2220(payload.Parameters, !enableOptimizeChargeGas)
	if err != nil {
		ts.log.Errorf("parse contract[%s] parameters error:%s", contractName, err)
		return errResult(result, fmt.Errorf(
			"parse tx[%s] contract[%s] parameters error:%s",
			payload.TxId,
			contractName,
			err.Error()),
		)
	}

	ts.log.Debugf("runVM => txSimContext.GetContractByName(`%s`) for tx `%v`", contractName, tx.GetPayload().TxId)
	contract, err := txSimContext.GetContractByName(contractName)
	if err != nil {
		ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
		return errResult(result, err)
	}
	if contract.RuntimeType != commonPb.RuntimeType_NATIVE && contract.RuntimeType != commonPb.RuntimeType_DOCKER_GO {
		byteCode, err = txSimContext.GetContractBytecode(contract.Name)
		if err != nil {
			ts.log.Errorf("Get contract bytecode by name[%s] error:%s", contract.Name, err)
			return errResult(result, err)
		}
	} else {
		ts.log.DebugDynamic(func() string {
			contractData, _ := json.Marshal(contract)
			return fmt.Sprintf("contract[%s] is a native contract, definition:%s",
				contractName, string(contractData))
		})
	}

	if ts.checkGasEnable() && !enableOptimizeChargeGas {
		accountMangerContract, pk, err = ts.getAccountMgrContractAndPk(txSimContext, tx, contract.Name, method)
		if err != nil {
			return result, specialTxType, err
		}

		// charge gas limit
		_, err = ts.chargeGasLimit(accountMangerContract, tx, txSimContext, contract.Name, method, pk, result)
		if err != nil {
			ts.log.Errorf("charge gas limit err is %v", err)
			result.Code = commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED
			result.Message = err.Error()
			result.ContractResult.Code = uint32(1)
			result.ContractResult.Message = err.Error()
			return result, specialTxType, err
		}
	}

	contractResultPayload, specialTxType, txStatusCode = ts.VmManager.RunContract(contract, method, byteCode,
		parameters, txSimContext, 0, tx.Payload.TxType)
	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	// refund gas
	if ts.checkGasEnable() {
		// check if this invoke needs charging gas
		if !ts.checkNativeFilter(contract.Name, method, tx, txSimContext.GetSnapshot()) {
			return result, specialTxType, err
		}

		// get tx's gas limit
		limit, err := getTxGasLimit(tx)
		if err != nil {
			ts.log.Errorf("getTxGasLimit error: %v", err)
			result.Message = err.Error()
			return result, specialTxType, err
		}

		// compare the gas used with gas limit
		if limit < contractResultPayload.GasUsed {
			err = fmt.Errorf("gas limit is not enough, [limit:%d]/[gasUsed:%d]",
				limit, contractResultPayload.GasUsed)
			ts.log.Error(err.Error())
			result.ContractResult.Code = uint32(commonPb.TxStatusCode_CONTRACT_FAIL)
			result.ContractResult.Message = err.Error()
			result.ContractResult.GasUsed = limit
			return result, specialTxType, err
		}
		if !enableOptimizeChargeGas {
			if _, err = ts.refundGas(accountMangerContract, tx, txSimContext, contractName, method, pk, result,
				contractResultPayload); err != nil {
				ts.log.Errorf("refund gas err is %v", err)
			}
		}
	}

	if txStatusCode == commonPb.TxStatusCode_SUCCESS {
		return result, specialTxType, nil
	}
	return result, specialTxType, errors.New(contractResultPayload.Message)
}

func (ts *TxScheduler) runVM2210(tx *commonPb.Transaction, txSimContext protocol.TxSimContext) (
	*commonPb.Result, protocol.ExecOrderTxType, error) {
	var (
		contractName          string
		method                string
		byteCode              []byte
		pk                    []byte
		specialTxType         protocol.ExecOrderTxType
		accountMangerContract *commonPb.Contract
		contractResultPayload *commonPb.ContractResult
		txStatusCode          commonPb.TxStatusCode
	)

	result := &commonPb.Result{
		Code: commonPb.TxStatusCode_SUCCESS,
		ContractResult: &commonPb.ContractResult{
			Code:    uint32(0),
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}
	payload := tx.Payload
	if payload.TxType != commonPb.TxType_QUERY_CONTRACT && payload.TxType != commonPb.TxType_INVOKE_CONTRACT {
		return errResult(result, fmt.Errorf("no such tx type: %s", tx.Payload.TxType))
	}

	contractName = payload.ContractName
	method = payload.Method
	parameters, err := ts.parseParameter2210(payload.Parameters)
	if err != nil {
		ts.log.Errorf("parse contract[%s] parameters error:%s", contractName, err)
		return errResult(result, fmt.Errorf(
			"parse tx[%s] contract[%s] parameters error:%s",
			payload.TxId,
			contractName,
			err.Error()),
		)
	}

	contract, err := txSimContext.GetContractByName(contractName)
	if err != nil {
		ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
		return errResult(result, err)
	}
	if contract.RuntimeType != commonPb.RuntimeType_NATIVE && contract.RuntimeType != commonPb.RuntimeType_DOCKER_GO {
		byteCode, err = txSimContext.GetContractBytecode(contractName)
		if err != nil {
			ts.log.Errorf("Get contract bytecode by name[%s] error:%s", contractName, err)
			return errResult(result, err)
		}
	} else {
		ts.log.DebugDynamic(func() string {
			contractData, _ := json.Marshal(contract)
			return fmt.Sprintf("contract[%s] is a native contract, definition:%s",
				contractName, string(contractData))
		})
	}

	accountMangerContract, pk, err = ts.getAccountMgrContractAndPk(txSimContext, tx, contractName, method)
	if err != nil {
		return result, specialTxType, err
	}

	// charge gas limit
	_, err = ts.chargeGasLimit(accountMangerContract, tx, txSimContext, contractName, method, pk, result)
	if err != nil {
		ts.log.Errorf("charge gas limit err is %v", err)
		result.Code = commonPb.TxStatusCode_GAS_BALANCE_NOT_ENOUGH_FAILED
		result.Message = err.Error()
		result.ContractResult.Code = uint32(1)
		result.ContractResult.Message = err.Error()
		return result, specialTxType, err
	}

	contractResultPayload, specialTxType, txStatusCode = ts.VmManager.RunContract(contract, method, byteCode,
		parameters, txSimContext, 0, tx.Payload.TxType)
	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	// refund gas
	_, err = ts.refundGas(accountMangerContract, tx, txSimContext, contractName, method, pk, result,
		contractResultPayload)
	if err != nil {
		ts.log.Errorf("refund gas err is %v", err)
	}

	if txStatusCode == commonPb.TxStatusCode_SUCCESS {
		return result, specialTxType, nil
	}
	return result, specialTxType, errors.New(contractResultPayload.Message)
}

func (ts *TxScheduler) parseParameter2220(
	parameterPairs []*commonPb.KeyValuePair,
	checkParamsNum bool) (map[string][]byte, error) {
	// verify parameters
	if checkParamsNum && len(parameterPairs) > protocol.ParametersKeyMaxCount {
		return nil, fmt.Errorf(
			"expect parameters length less than %d, but got %d",
			protocol.ParametersKeyMaxCount,
			len(parameterPairs),
		)
	}
	parameters := make(map[string][]byte, 16)
	for i := 0; i < len(parameterPairs); i++ {
		key := parameterPairs[i].Key
		value := parameterPairs[i].Value
		if len(key) > protocol.DefaultMaxStateKeyLen {
			return nil, fmt.Errorf(
				"expect key length less than %d, but got %d",
				protocol.DefaultMaxStateKeyLen,
				len(key),
			)
		}
		match := ts.keyReg.MatchString(key)
		if !match {
			return nil, fmt.Errorf(
				"expect key no special characters, but got key:[%s]. letter, number, dot and underline are allowed",
				key,
			)
		}
		if len(value) > int(protocol.ParametersValueMaxLength) {
			return nil, fmt.Errorf(
				"expect value length less than %d, but got %d",
				protocol.ParametersValueMaxLength,
				len(value),
			)
		}

		parameters[key] = value
	}
	return parameters, nil
}

func (ts *TxScheduler) parseParameter2210(parameterPairs []*commonPb.KeyValuePair) (map[string][]byte, error) {
	// verify parameters
	if len(parameterPairs) > protocol.ParametersKeyMaxCount {
		return nil, fmt.Errorf(
			"expect parameters length less than %d, but got %d",
			protocol.ParametersKeyMaxCount,
			len(parameterPairs),
		)
	}
	parameters := make(map[string][]byte, 16)
	for i := 0; i < len(parameterPairs); i++ {
		key := parameterPairs[i].Key
		value := parameterPairs[i].Value
		if len(key) > protocol.DefaultMaxStateKeyLen {
			return nil, fmt.Errorf(
				"expect key length less than %d, but got %d",
				protocol.DefaultMaxStateKeyLen,
				len(key),
			)
		}
		match := ts.keyReg.MatchString(key)
		if !match {
			return nil, fmt.Errorf(
				"expect key no special characters, but got key:[%s]. letter, number, dot and underline are allowed",
				key,
			)
		}
		if len(value) > int(protocol.ParametersValueMaxLength) {
			return nil, fmt.Errorf(
				"expect value length less than %d, but got %d",
				protocol.ParametersValueMaxLength,
				len(value),
			)
		}

		parameters[key] = value
	}
	return parameters, nil
}

func (ts *TxScheduler) getContractFromCache(txSimContext protocol.TxSimContext,
	contractName string) (*commonPb.Contract, error) {
	var contract *commonPb.Contract
	var err error
	// if contract exists in cache, assign to contract
	if ct, ok := ts.contractCache.Load(contractName); ok {
		if contract, ok = ct.(*commonPb.Contract); !ok {
			err = errors.New("failed to transfer contract from interface to struct")
			ts.log.Error(err)
			return nil, err
		}
	} else {
		// contract not exists in cache, use single flight to get contract
		ct, err, _ = sf.Do(contractName, func() (interface{}, error) {
			var ctTmp *commonPb.Contract
			ctTmp, err = txSimContext.GetContractByName(contractName)
			if err != nil {
				ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
				return nil, err
			}
			// store to contract cache after get contract
			ts.contractCache.Store(contractName, ctTmp)
			return ctTmp, nil
		})

		if err != nil {
			return nil, err
		}

		if contract, ok = ct.(*commonPb.Contract); !ok {
			err = errors.New("failed to transfer contract from interface to struct")
			ts.log.Error(err)
			return nil, err
		}
	}
	return contract, nil
}

func (ts *TxScheduler) getContractBytecode(txSimContext protocol.TxSimContext,
	contract *commonPb.Contract) ([]byte, error) {
	if contract.RuntimeType != commonPb.RuntimeType_NATIVE &&
		contract.RuntimeType != commonPb.RuntimeType_DOCKER_GO &&
		contract.RuntimeType != commonPb.RuntimeType_GO &&
		contract.RuntimeType != commonPb.RuntimeType_DOCKER_JAVA {
		byteCode, err := txSimContext.GetContractBytecode(contract.Name)
		if err != nil {
			ts.log.Errorf("Get contract bytecode by name[%s] error:%s", contract.Name, err)
			return nil, err
		}
		return byteCode, nil
	}
	ts.log.DebugDynamic(func() string {
		contractData, _ := json.Marshal(contract)
		return fmt.Sprintf("contract[%s] is a native contract, definition:%s",
			contract.Name, string(contractData))
	})
	return nil, nil
}

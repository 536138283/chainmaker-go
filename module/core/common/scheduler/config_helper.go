package scheduler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"chainmaker.org/chainmaker/common/v3/crypto"
	"chainmaker.org/chainmaker/utils/v3"

	"chainmaker.org/chainmaker/pb-go/v3/syscontract"

	configPb "chainmaker.org/chainmaker/pb-go/v3/config"

	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/protocol/v3"
)

// VerifyOptimizeChargeGasTx verify gas tx
// @param block
// @param snapshot
// @return error
func VerifyOptimizeChargeGasTx(block *commonPb.Block, snapshot protocol.Snapshot) error {
	// gas to charge from validator
	gasCalc := make(map[string]uint64, 24)
	// gas to charge from proposer
	gasNeedToCharge := make(map[string]uint64, 24)
	chainCfg, err := snapshot.GetBlockchainStore().GetLastChainConfig()
	if err != nil {
		return fmt.Errorf("GetLastChainConfig error: %v", err)
	}

	// 软分叉处理，v240之后使用coinbase实现，不再有GasTx
	var contractName, methodName string
	blockVersion := block.GetHeader().BlockVersion
	if blockVersion >= blockVersion3000000 {
		contractName = syscontract.SystemContract_COINBASE.String()
		methodName = syscontract.CoinbaseFunction_RUN_COINBASE.String()
	} else {
		contractName = syscontract.SystemContract_ACCOUNT_MANAGER.String()
		methodName = syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String()
	}

	found := false
	for _, tx := range block.Txs {
		if tx.Payload.ContractName == contractName && tx.Payload.Method == methodName {
			found = true
			if blockVersion >= blockVersion3000000 {
				senders, err1 := getSenders(tx.Payload.Parameters)
				if err1 != nil {
					return err1
				}
				for k, v := range senders {
					total, err2 := strconv.ParseUint(string(v), 10, 64)
					if err2 != nil {
						return fmt.Errorf("ParseUint error: %v", err2)
					}
					gasNeedToCharge[k] = total
				}
			} else {
				for _, kv := range tx.Payload.Parameters {
					total, err2 := strconv.ParseUint(string(kv.Value), 10, 64)
					if err2 != nil {
						return fmt.Errorf("ParseUint error: %v", err2)
					}
					gasNeedToCharge[kv.Key] = total
				}
			}
		} else {
			gasUsed := tx.Result.ContractResult.GasUsed
			pk, err2 := getPayerPkFromTx(tx, snapshot)
			if err2 != nil {
				return fmt.Errorf("getPayerPkFromTx error: %v", err2)
			}

			// convert the public key to `ZX` or `CM` or `EVM` address
			address, err2 := publicKeyToAddress(pk, chainCfg)
			if err2 != nil {
				return fmt.Errorf("publicKeyToAddress failed: err = %v", err)
			}
			if totalGas, exists := gasCalc[address]; exists {
				gasCalc[address] = totalGas + gasUsed
			} else {
				gasCalc[address] = gasUsed
			}
		}
	}

	if !found {
		return fmt.Errorf("charge gas tx is missing")
	}
	// compare gasCalc and gasNeedToCharge
	if len(gasCalc) != len(gasNeedToCharge) {
		return fmt.Errorf("gas need to charging is not correct, expect %v account, got %v account",
			len(gasCalc), len(gasNeedToCharge))
	}

	for addr, totalGasCalc := range gasCalc {
		if totalGasNeedToCharge, exists := gasNeedToCharge[addr]; !exists {
			return fmt.Errorf("missing some account to charge gas => `%v`", addr)
		} else if totalGasCalc != totalGasNeedToCharge {
			return fmt.Errorf("gas to charge error for address `%v`, expect %v, got %v",
				addr, totalGasCalc, totalGasNeedToCharge)
		}
	}

	return nil
}

// publicKeyToAddress: generate address from public key, according to chainconfig parameter
func publicKeyToAddress(pk crypto.PublicKey, chainCfg *configPb.ChainConfig) (string, error) {

	publicKeyString, err := utils.PkToAddrStr(pk, chainCfg.Vm.AddrType, crypto.HashAlgoMap[chainCfg.Crypto.Hash])
	if err != nil {
		return "", err
	}

	if chainCfg.Vm.AddrType == configPb.AddrType_ZXL {
		publicKeyString = "ZX" + publicKeyString
	}
	return publicKeyString, nil
}

func getSenders(parameters []*commonPb.KeyValuePair) (map[string][]byte, error) {
	for _, kv := range parameters {
		if kv.Key == chargeGasVmForMultiAccountParameterKey {
			senders := make(map[string][]byte)
			err := json.Unmarshal(kv.Value, &senders)
			if err != nil {
				return nil, fmt.Errorf("senders unmarshal error")
			}
			return senders, nil
		}
	}
	return nil, fmt.Errorf("%s not found", chargeGasVmForMultiAccountParameterKey)
}

func getMultiSignEnableManualRun(chainConfig *configPb.ChainConfig) bool {
	if chainConfig.Vm == nil {
		return false
	} else if chainConfig.Vm.Native == nil {
		return false
	} else if chainConfig.Vm.Native.Multisign == nil {
		return false
	}

	return chainConfig.Vm.Native.Multisign.EnableManualRun
}

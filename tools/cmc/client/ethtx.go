package client

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/tools/cmc/types"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v3/ethbase"
	"chainmaker.org/chainmaker/common/v3/evmutils/abi"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

func sendRawTransactionCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sendrawtransaction [hexrawtx]",
		Short: "send raw transaction",
		Long:  "send signed raw transaction",
		RunE: func(_ *cobra.Command, args []string) error {
			return sendRawTransaction(args[0])
		},
		Args: cobra.ExactArgs(1),
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagSyncResult, flagAbiFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}
func sendRawTransaction(rawTxHex string) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	rawTx, err := ethbase.HexToBytes(rawTxHex)
	if err != nil {
		return err
	}
	req := &commonPb.TxRequest{Payload: &commonPb.Payload{TxType: commonPb.TxType_ETH_TX}}
	req.Payload.Parameters = []*commonPb.KeyValuePair{{
		Key:   "rawtx",
		Value: rawTx,
	}}
	req.Payload.TxId = hex.EncodeToString(ethbase.Keccak256(rawTx))
	resp, err := client.SendTxRequest(req, timeout, syncResult)
	if err != nil {
		fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
		return err
	}
	if !syncResult {
		util.PrintPrettyJson(resp)
		return nil
	}
	//处理返回的结果，ABI解包
	var contractAbi *abi.ABI

	if abiFilePath != "" { // abi file path 非空 意味着调用的是EVM合约
		abiBytes, err := ioutil.ReadFile(abiFilePath)
		if err != nil {
			return err
		}

		contractAbi, err = abi.JSON(bytes.NewReader(abiBytes))
		if err != nil {
			return err
		}

	}
	var output interface{}
	if contractAbi != nil && resp.ContractResult != nil && resp.ContractResult.Result != nil {
		unpackedData, err := contractAbi.Unpack(method, resp.ContractResult.Result)
		if err != nil {
			fmt.Println(err)
			return err
		}
		output = types.EvmTxResponse{
			TxResponse: resp,
			ContractResult: &types.EvmContractResult{
				ContractResult: resp.ContractResult,
				Result:         fmt.Sprintf("%v", unpackedData),
			},
		}
	} else {
		if respResultToString {
			output = util.RespResultToString(resp)
		} else {
			output = resp
		}
	}
	util.PrintPrettyJson(output)
	return nil
}

func estimateGasCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate-gas",
		Short: "Estimate Gas",
		Long:  "Estimate Gas",
		RunE: func(_ *cobra.Command, args []string) error {
			data, _ := hex.DecodeString(ethData)
			return estimateGas(ethFrom, ethTo, ethGas, ethGasPrice, ethValue, data)
		},
		Args: cobra.ExactArgs(0),
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagSyncResult, flagEthFrom, flagEthTo, flagEthGas, flagEthGasPrice, flagEthValue, flagEthData,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagEthTo)

	return cmd
}

func estimateGas(from, to string, gas uint64, gasPrice uint64, value uint64, data []byte) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	var fromAddr *ethbase.Address
	if len(from) > 0 {
		f, err1 := ethbase.ParseAddress(from)
		if err1 != nil {
			return fmt.Errorf("invalid address [from:%s]", from)
		}
		fromAddr = &f
	}
	toAddr, err := ethbase.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid address [to:%s]", to)
	}
	gasPrice256 := ethbase.NewSafeUint256FromUint64(gasPrice)
	value256 := ethbase.NewSafeUint256FromUint64(value)
	gasUsed, err := client.EthEstimateGas(fromAddr, toAddr, gas, gasPrice256, value256, data)
	if err != nil {
		return fmt.Errorf("query contract failed, %s", err.Error())
	}

	fmt.Printf("Gas used:\t%d\n", gasUsed)
	return nil
}

func callCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Call",
		Long:  "Call",
		RunE: func(_ *cobra.Command, args []string) error {
			data, _ := hex.DecodeString(ethData)
			return call(ethFrom, ethTo, ethGas, ethGasPrice, ethValue, data)
		},
		Args: cobra.ExactArgs(0),
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagSyncResult, flagEthFrom, flagEthTo, flagEthGas, flagEthGasPrice, flagEthValue, flagEthData,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagEthFrom)
	cmd.MarkFlagRequired(flagEthTo)

	return cmd
}

func call(from string, to string, gas uint64, price uint64, value uint64, data []byte) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	fromAddr, err := ethbase.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("invalid address [from:%s]", from)
	}
	toAddr, err := ethbase.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid address [to:%s]", to)
	}
	gasPrice256 := ethbase.NewSafeUint256FromUint64(price)
	value256 := ethbase.NewSafeUint256FromUint64(value)
	resp, err := client.EthCall(fromAddr, toAddr, gas, gasPrice256, value256, data)

	if err != nil {
		return fmt.Errorf("query contract failed, %s", err.Error())
	}
	output, err := prettyjson.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

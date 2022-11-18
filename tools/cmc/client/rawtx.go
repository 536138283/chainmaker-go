package client

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/tools/cmc/types"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/ethbase"
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
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
	rawTx, err := ethbase.DecodeHex(rawTxHex)
	if err != nil {
		return err
	}
	req := &commonPb.TxRequest{Payload: &commonPb.Payload{TxType: commonPb.TxType_ETH_TX_LegacyTxType}}
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

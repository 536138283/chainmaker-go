package gas

import (
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func setContractMethodPayerCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-payer [contract_name] [method] [payer_address]",
		Short: "set gas payer for contract method",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			// 建立链接
			client, err := util.CreateChainClientWithConfPath(sdkConfPath, false)
			if err != nil {
				return err
			}
			defer client.Stop()

			payerKeyPem, err := ioutil.ReadFile(payerKeyFilePath)
			if err != nil {
				return err
			}
			payerCertPem, err := ioutil.ReadFile(payerCrtFilePath)
			if err != nil {
				return err
			}
			privateKey, err := asym.PrivateKeyFromPEM(payerKeyPem, nil)
			if err != nil {
				return err
			}

			// 构建参数
			message := contractName
			message += ":" + method
			message += ":" + address
			message += ":" + uuid.GetUUID()
			signature, err := sdkutils.SignPayloadBytesWithHashType(
				privateKey,
				client.GetHashType(),
				[]byte(message))
			if err != nil {
				return err
			}

			memberInfo := payerCertPem
			var memberType acPb.MemberType
			if client.GetAuthType() == sdk.PermissionedWithCert {
				memberType = acPb.MemberType_CERT
			} else if client.GetAuthType() == sdk.PermissionedWithKey {
				memberType = acPb.MemberType_PUBLIC_KEY
			} else if client.GetAuthType() == sdk.Public {
				memberType = acPb.MemberType_PUBLIC_KEY
			}
			endorsement := sdkutils.NewEndorserWithMemberType(payerOrgId, memberInfo, memberType, signature)
			endorsementBytes, err := proto.Marshal(endorsement)
			if err != nil {
				return err
			}

			var params []*common.KeyValuePair
			params = append(params, &common.KeyValuePair{
				Key:   "ENDORSEMENT_ENTRY",
				Value: endorsementBytes,
			})

			// 构建 payload
			var payload *common.Payload
			if multiSign {
				payload = client.CreatePayload(
					"", common.TxType_INVOKE_CONTRACT,
					syscontract.SystemContract_MULTI_SIGN.String(),
					syscontract.MultiSignFunction_REQ.String(),
					params,
					0, nil)
			} else {
				payload = client.CreatePayload(
					"", common.TxType_INVOKE_CONTRACT,
					syscontract.SystemContract_ACCOUNT_MANAGER.String(),
					syscontract.GasAccountFunction_SET_METHOD_PAYER.String(),
					params,
					0, &common.Limit{GasLimit: gasLimit})
			}

			// 产生 Request
			request, err := client.GenerateTxRequest(payload, nil)
			if err != nil {
				return err
			}

			// 发送 Request, 读取 Response
			resp, err := client.SendTxRequest(request, -1, true)
			if err != nil {
				return err
			}

			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagContractName, flagMethod, flagAddress, flagMultiSign,
		flagPayerKeyFilePath, flagPayerCrtFilePath, flagPayerOrgId,
		flagSdkConfPath, flagGasLimit,
	})

	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

func unsetContractMethodPayerCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset-payer [contract_name] [method]",
		Short: "clear the gas payer setting for contract's method",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			// 建立链接
			client, err := util.CreateChainClientWithConfPath(sdkConfPath, false)
			if err != nil {
				return err
			}
			defer client.Stop()

			payerKeyPem, err := ioutil.ReadFile(payerKeyFilePath)
			if err != nil {
				return err
			}
			payerCertPem, err := ioutil.ReadFile(payerCrtFilePath)
			if err != nil {
				return err
			}
			privateKey, err := asym.PrivateKeyFromPEM(payerKeyPem, nil)
			if err != nil {
				return err
			}

			// 构建参数
			message := contractName
			message += ":" + method
			message += ":" + address
			message += ":" + uuid.GetUUID()
			signature, err := sdkutils.SignPayloadBytesWithHashType(
				privateKey,
				client.GetHashType(),
				[]byte(message))
			if err != nil {
				return err
			}

			memberInfo := payerCertPem
			var memberType acPb.MemberType
			if client.GetAuthType() == sdk.PermissionedWithCert {
				memberType = acPb.MemberType_CERT
			} else if client.GetAuthType() == sdk.PermissionedWithKey {
				memberType = acPb.MemberType_PUBLIC_KEY
			} else if client.GetAuthType() == sdk.Public {
				memberType = acPb.MemberType_PUBLIC_KEY
			}
			endorsement := sdkutils.NewEndorserWithMemberType(payerOrgId, memberInfo, memberType, signature)
			endorsementBytes, err := proto.Marshal(endorsement)
			if err != nil {
				return err
			}

			var params []*common.KeyValuePair
			params = append(params, &common.KeyValuePair{
				Key:   "ENDORSEMENT_ENTRY",
				Value: endorsementBytes,
			})

			// 构建 payload
			var payload *common.Payload
			if multiSign {
				payload = client.CreatePayload(
					"", common.TxType_INVOKE_CONTRACT,
					syscontract.SystemContract_MULTI_SIGN.String(),
					syscontract.MultiSignFunction_REQ.String(),
					params,
					0, nil)
			} else {
				payload = client.CreatePayload(
					"", common.TxType_INVOKE_CONTRACT,
					syscontract.SystemContract_ACCOUNT_MANAGER.String(),
					syscontract.GasAccountFunction_UNSET_METHOD_PAYER.String(),
					params,
					0, &common.Limit{GasLimit: gasLimit})
			}

			// 产生 Request
			request, err := client.GenerateTxRequest(payload, nil)
			if err != nil {
				return err
			}

			// 发送 Request, 读取 Response
			resp, err := client.SendTxRequest(request, -1, true)
			if err != nil {
				return err
			}

			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagContractName, flagMethod, flagAddress, flagMultiSign,
		flagSdkConfPath,
	})

	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func queryContractMethodPayerCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-payer [contract_name] [method]",
		Short: "query the payer setting for the contract's method",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClientWithConfPath(sdkConfPath, false)
			if err != nil {
				return err
			}
			defer client.Stop()

			var params []*common.KeyValuePair
			contractName = strings.TrimSpace(contractName)
			if contractName != "" {
				params = append(params, &common.KeyValuePair{
					Key:   "CONTRACT_NAME",
					Value: []byte(contractName),
				})
			}
			method = strings.TrimSpace(method)
			if method != "" {
				params = append(params, &common.KeyValuePair{
					Key:   "METHOD",
					Value: []byte(method),
				})
			}

			// 构建 payload
			payload := client.CreatePayload(
				"", common.TxType_INVOKE_CONTRACT,
				syscontract.SystemContract_ACCOUNT_MANAGER.String(),
				syscontract.GasAccountFunction_GET_METHOD_PAYER.String(),
				params,
				0, &common.Limit{GasLimit: gasLimit})

			// 产生 Request
			request, err := client.GenerateTxRequest(payload, nil)
			if err != nil {
				return err
			}

			// 发送 Request, 读取 Response
			resp, err := client.SendTxRequest(request, -1, true)
			if err != nil {
				return err
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagContractName, flagMethod, flagAddress,
		flagSdkConfPath,
	})

	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func queryTxPayerCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-tx-payer [txId]",
		Short: "query the payer address of the tx",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClientWithConfPath(sdkConfPath, false)
			if err != nil {
				return err
			}
			defer client.Stop()

			var params []*common.KeyValuePair
			contractName = strings.TrimSpace(contractName)
			if contractName != "" {
				params = append(params, &common.KeyValuePair{
					Key:   "CONTRACT_NAME",
					Value: []byte(contractName),
				})
			}
			method = strings.TrimSpace(method)
			if method != "" {
				params = append(params, &common.KeyValuePair{
					Key:   "METHOD",
					Value: []byte(method),
				})
			}

			// 构建 payload
			payload := client.CreatePayload(
				"", common.TxType_INVOKE_CONTRACT,
				syscontract.SystemContract_ACCOUNT_MANAGER.String(),
				syscontract.GasAccountFunction_GET_TX_PAYER.String(),
				params,
				0, &common.Limit{GasLimit: gasLimit})

			// 产生 Request
			request, err := client.GenerateTxRequest(payload, nil)
			if err != nil {
				return err
			}

			// 发送 Request, 读取 Response
			resp, err := client.SendTxRequest(request, -1, true)
			if err != nil {
				return err
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagContractName, flagMethod, flagAddress,
		flagSdkConfPath,
	})

	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/11/24
package native_test

import (
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 证书添加，个人添加自己的证书
func TestCertAdd(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERT_ADD.String()})
	processResults(resp, err)
}

// 证书的删除（管理员操作）
func TestCertDelete(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes_1",
			Value: "de536ef8c323ae708586f9486bc3fe8bbba6452ff940a6e69de00db5159e5b1e",
		},
	}
	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_DELETE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQuery(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes",
			Value: "b297d4e74ba0a88f9d154d63e53f2bf57116e09b6b8d2a718c24e59175f74cbe",
		},
	}
	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQueryWithCertId(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight in TestCertQueryWithCertId============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes_2",
			Value: "b297d4e74ba0a88f9d154d63e53f2bf57116e09b6b8d2a718c24e59175f74cbe",
		},
	}

	sk, _ := native.GetUserSK(1)
	resp, err := native.QueryRequestWithCertID(sk, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书冻结
func TestCertFrozen(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "certs", // org1 admin sign cert
			Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi6gAwIBAgIDDk05MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZAxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ8wDQYDVQQLEwZjbGllbnQxKzApBgNVBAMTImNsaWVudDEudGxzLnd4LW9yZzEu\nY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQPzwfs7BU+\nK0F/Y2MIfZlEzQv2Tdyxb2ermoVvIA5Kwz6mmLSqsVX6ZxBwYQ/gf9VMzFZkKadV\nntrl34lC3jY5o3sweTAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgwBgYEVR0lADAp\nBgNVHQ4EIgQgnb+Rban1rokuCZHYKzDhNm/nqnam4YdneDyfo8CSHzswKwYDVR0j\nBCQwIoAgNSQ/cRy5t8Q1LpMfcMVzMfl0CcLZ4Pvf7BxQX9sQiWcwCgYIKoZIzj0E\nAwIDRwAwRAIgGYzsN0+mqMOawe0T6eicuW1mlwQCu2Qt0Y8IDbJjuQYCIB85HFCd\nUQ908a46G/eQ8YMeQApBkVhFDBN82soA3jQA\n-----END CERTIFICATE-----\n",
		},
		//{
		//	Key:   "certs", // other admin sign cert
		//	Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDCtpUMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIxMDgyNjAyMjIxM1oXDTI2\nMDgyNTAyMjIxM1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABECzNJVm2ew1\nAgSXpcxN4Ia5kbZsX/to68jgTNIwgjfkfXht6M854YjtCw0Hr9XsUEa/7rdTULNR\nqu6TSLdg2d+jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCCP81zjXAdtdND0JFQl55lLNeIbaQuYB+qyNzoTHogtZzArBgNVHSME\nJDAigCBLAARy/poGa+Z/HntGrZGZgGSjBzo5sy7UMmCrCd9r9TAKBggqhkjOPQQD\nAgNIADBFAiEAmS0Z6TYaChL7ywnHsNYYMP76OSPjKxC4nh2fZLPK3CYCIHoi/TaI\nGoWu8fLp0tig3auDal15f8um9wk/UzZcblCA\n-----END CERTIFICATE-----\n",
		//},
		//{
		//	Key:   "certs", // ca
		//	Value: "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n-----END CERTIFICATE-----\n",
		//},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_FREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书解冻
func TestCertUnfrozen(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "certs", // org1 admin sign cert
			Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDAwGbMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABORqoYNAw8ax\n9QOD94VaXq1dCHguarSKqAruEI39dRkm8Vu2gSHkeWlxzvSsVVqoN6ATObi2ZohY\nKYab2s+/QA2jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCDZOtAtHzfoZd/OQ2Jx5mIMgkqkMkH4SDvAt03yOrRnBzArBgNVHSME\nJDAigCA1JD9xHLm3xDUukx9wxXMx+XQJwtng+9/sHFBf2xCJZzAKBggqhkjOPQQD\nAgNIADBFAiEAiGjIB8Wb8mhI+ma4F3kCW/5QM6tlxiKIB5zTcO5E890CIBxWDICm\nAod1WZHJajgnDQ2zEcFF94aejR9dmGBB/P//\n-----END CERTIFICATE-----\n",
		},
		//{
		//	Key:   "certs", // other admin sign cert
		//	Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDCtpUMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIxMDgyNjAyMjIxM1oXDTI2\nMDgyNTAyMjIxM1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABECzNJVm2ew1\nAgSXpcxN4Ia5kbZsX/to68jgTNIwgjfkfXht6M854YjtCw0Hr9XsUEa/7rdTULNR\nqu6TSLdg2d+jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCCP81zjXAdtdND0JFQl55lLNeIbaQuYB+qyNzoTHogtZzArBgNVHSME\nJDAigCBLAARy/poGa+Z/HntGrZGZgGSjBzo5sy7UMmCrCd9r9TAKBggqhkjOPQQD\nAgNIADBFAiEAmS0Z6TYaChL7ywnHsNYYMP76OSPjKxC4nh2fZLPK3CYCIHoi/TaI\nGoWu8fLp0tig3auDal15f8um9wk/UzZcblCA\n-----END CERTIFICATE-----\n",
		//},
		//{
		//	Key:   "certs", // ca
		//	Value: "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n-----END CERTIFICATE-----\n",
		//},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_UNFREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书解冻
func TestCertUnfrozenWithCertHash(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes_3",
		Value: "ae052a0deeffe50ba2b447ed43b77c505dd0cc8c8dc918ce3dbb51073d874729,09ff34fafd2b97c8e9c7e05704b075d90cb7fee93cd2e4234e71cee6df0a88e6",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_UNFREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书吊销
func TestCertRevocation(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Println("============ get chain config by blockHeight in TestCertRevocation============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "cert_crl",
		// 多个就换行就行
		Value: "-----BEGIN CRL-----\nMIIBVjCB/AIBATAKBggqgRzPVQGDdTCBgzELMAkGA1UEBhMCQ04xEDAOBgNVBAgT\nB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxHzAdBgNVBAoTFnd4LW9yZzEuY2hh\naW5tYWtlci5vcmcxCzAJBgNVBAsTAmNhMSIwIAYDVQQDExljYS53eC1vcmcxLmNo\nYWlubWFrZXIub3JnFw0yMTAxMTMwNjQ4MzhaFw0yMTAxMTMxMDQ4MzhaMBYwFAID\nDn50Fw0yMjAxMTIwMzM4MjJaoC8wLTArBgNVHSMEJDAigCAsQ4wyJIOuunNAHBqt\nESXwwBsY1fTkz7+vyHiD211y2zAKBggqgRzPVQGDdQNJADBGAiEA/ksRnjkjxpia\nfnOSCk557rPYWBFBxyYoyAbb22L39zwCIQCJsIiMNThs8VJN2MKaEeOSSSD1Z/0i\nrjsVWvt1I3nDpQ==\n-----END CRL-----\n",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_REVOKE.String(), Pairs: pairs})
	processResults(resp, err)
}

func processResults(resp *commonPb.TxResponse, err error) {
	if err == nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println("WARN: client.call err: deadline")
		return
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
}

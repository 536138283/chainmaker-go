/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	bccrypto "chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/localconf"
	logger2 "chainmaker.org/chainmaker-go/logger"
	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/stretchr/testify/require"
)

const (
	chainId1 = "chain1"
	version  = "v1.0.0"
	org1Name = "wx-org1.chainmaker.org"
	org2Name = "wx-org2.chainmaker.org"
	org3Name = "wx-org3.chainmaker.org"
	org4Name = "wx-org4.chainmaker.org"
	org5Name = "wx-org5.chainmaker.org"

	msg = "Winter is coming."

	tempOrg1KeyFileName  = "org1.key"
	tempOrg1CertFileName = "org1.crt"
)

var chainConf = &config.ChainConfig{
	ChainId:  chainId1,
	Version:  version,
	AuthType: string(IdentityMode),
	Sequence: 0,
	Crypto: &config.CryptoConfig{
		Hash: bccrypto.CRYPTO_ALGO_SHA256,
	},
	Block: nil,
	Core:  nil,
	Consensus: &config.ConsensusConfig{
		Type: 0,
		Nodes: []*config.OrgConfig{{
			OrgId:  org1Name,
			NodeId: nil,
		}, {
			OrgId:  org2Name,
			NodeId: nil,
		}, {
			OrgId:  org3Name,
			NodeId: nil,
		}, {
			OrgId:  org4Name,
			NodeId: nil,
		},
		},
		ExtConfig: nil,
	},
	TrustRoots: []*config.TrustRootConfig{{
		OrgId: org1Name,
		Root: `-----BEGIN CERTIFICATE-----
MIICnjCCAkSgAwIBAgIDChC/MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAS2sF1wu8c8Uox3CR67
Yfp6gZj/X5oXJx0JiVOryLO+0BmlpMDymbKejNeH3cOH55zPVkr4Fpq0yhSBEmEA
RZq8o4GWMIGTMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MCkGA1Ud
DgQiBCDsHUjFfdIYla+6dtUW2FzP5/pJam+fmC1O9J5LUBTjTTBFBgNVHREEPjA8
gg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmcxLmNoYWlubWFr
ZXIub3JnhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIEn4Mbn2hqT5HfwmCDnrM4hI
nTuStQZziqeyzEAwShRqAiEA8iGF3PidR6Zi/2EzOQUKjdFK9sO7aXsSb+PbFVEE
nx8=
-----END CERTIFICATE-----`,
	}, {
		OrgId: org2Name,
		Root: `-----BEGIN CERTIFICATE-----
MIICnTCCAkSgAwIBAgIDC8pwMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASjk2uzvcGL5TvKDSBe
3dWYSzhxbrbTr/P70iv0sqfK4Ls1xWMD03hGO5egIPe2M0ehEQA0Z9SKZ0LWxeBz
Z2dIo4GWMIGTMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MCkGA1Ud
DgQiBCARrvELOzQ+siChyC/9a9w8JA7+z3mTI6T9D24yLu3w0zBFBgNVHREEPjA8
gg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmcyLmNoYWlubWFr
ZXIub3JnhwR/AAABMAoGCCqGSM49BAMCA0cAMEQCICXT/aFsHoKRljbdNMn2KZs2
/ak8fAUHqBmdVmCAOZUFAiBrZ/owWEbQYD52v7na4ssanJPtI862K0ZotG5xctJR
cA==
-----END CERTIFICATE-----
`,
	}, {
		OrgId: org3Name,
		Root: `-----BEGIN CERTIFICATE-----
MIICnTCCAkSgAwIBAgIDB8hdMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzMuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARSg4clzT+oac4mfINk
Pfv6yk0F1UQMHAYKtnSZPjSKOqWce9YxF5rSE4m4WTWxzWNbWJi853y8XjUcRRBV
ANXgo4GWMIGTMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MCkGA1Ud
DgQiBCD4ka4nrMDDyFEHagxtH5R55316xMe8JsjJ19RPrF0JlzBFBgNVHREEPjA8
gg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmczLmNoYWlubWFr
ZXIub3JnhwR/AAABMAoGCCqGSM49BAMCA0cAMEQCIBm5QLNM9oCmtpFgOXgHOodo
7WHpUqwB42qROt+zXwOEAiBb/Bkibt2CKFJ2CAr9x34uVnWyxem4RsmkDMWNnJcu
qQ==
-----END CERTIFICATE-----`,
	}, {
		OrgId: org4Name,
		Root: `-----BEGIN CERTIFICATE-----
MIICnzCCAkSgAwIBAgIDCCwjMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc0LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc0LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzQuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7yaF+mlca+00qgJnI
EUB4/nOZ2BkEmmxvp3bD3ar/4ZNDBrQapOiripS/NQSqn+G/yvVVGMdIF8abWklJ
KWAyo4GWMIGTMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MCkGA1Ud
DgQiBCDVjf2ac4jraYfvInytZgCuTZS7nhBCM5zzDH/SCOVlxzBFBgNVHREEPjA8
gg5jaGFpbm1ha2VyLm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmc0LmNoYWlubWFr
ZXIub3JnhwR/AAABMAoGCCqGSM49BAMCA0kAMEYCIQDZ/Bz+gPkPK8MIwYOzDCCb
M5QWQ/Q4z3pWgejYZtNFWQIhAJCLrs87Sk0dcC7fi154Ux/P1WmwJBrubnSpFAj1
cfGT
-----END CERTIFICATE-----`,
	},
	},
}

type certificatePair struct {
	certificate string
	sk          string
}
type orgInfo struct {
	orgId         string
	consensusNode certificatePair
	commonNode    certificatePair
	admin         certificatePair
	client        certificatePair
}
type ac struct {
	acInst        protocol.AccessControlProvider
	consensusNode protocol.SigningMember
	commonNode    protocol.SigningMember
	admin         protocol.SigningMember
	client        protocol.SigningMember
}

var orgList = map[string]orgInfo{
	org1Name: {
		orgId: org1Name,
		consensusNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICsDCCAlegAwIBAgIDC6UdMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmcxLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
ekr4ywSFEjaQrY/oUXujFX30HgnGGCsP/jKJknvTdAq2RmtBOrwCq8mC823eLtd0
7YJn2Vb18p9pXb+mQK7AbaOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIE
IMy7a2as+Os5jYCCHOZgwkwTy/27tPVdZO4XDEBVJOC8MCsGA1UdIwQkMCKAIOwd
SMV90hiVr7p21RbYXM/n+klqb5+YLU70nktQFONNMC8GC4EnWI9kCx6PZAsEBCAz
NDcyZGU0NTZlNzY0MzgwODUxYmQ3MjRlZTI4YzhiMDAKBggqhkjOPQQDAgNHADBE
AiAYlrlGzFzpZjJXcfbXJiMJDbfZyqltNnblj3VE3kg5KQIgQtcF98Z5aDxiQC6y
EDLJxURqwDW0C+fHGELVMvXjIJQ=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILFR2gF5nXvHdkLwdOi9VKbw6lu03gxu+Tu3EXpN6EdroAoGCCqGSM49
AwEHoUQDQgAEekr4ywSFEjaQrY/oUXujFX30HgnGGCsP/jKJknvTdAq2RmtBOrwC
q8mC823eLtd07YJn2Vb18p9pXb+mQK7AbQ==
-----END EC PRIVATE KEY-----`,
		},
		commonNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICqzCCAlGgAwIBAgIDCkkiMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEE8/W38/F
ICFyi6YKDetGpxBbjAQXLzEATY5n0QLvkzB5woXuxYL5GQOHMnSxNpLmA3oMcUXu
WAnYCZO36vca6aOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIEICvh68VZ
iYFQ0FrTik+ZKs230Xe7uKEF4m6zcZ7+k/+MMCsGA1UdIwQkMCKAIOwdSMV90hiV
r7p21RbYXM/n+klqb5+YLU70nktQFONNMC8GC4EnWI9kCx6PZAsEBCA2MzJjYTZh
MDkyYzU0ZGUwYjZhZDY5NjZkN2QxNWIyNzAKBggqhkjOPQQDAgNIADBFAiAnX2gF
VzHQVADjq7QT9Bs+pbCvbFgmfkSovc22rt7jQAIhAMIzd9yaog2DUg8iadqVzg7U
kbP9iWGJHkusrGiDu2EM
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIjtCqOmyNi82UBe1nGIrbQkMwMgRe60x1FO4xBeVoXcoAoGCCqGSM49
AwEHoUQDQgAEE8/W38/FICFyi6YKDetGpxBbjAQXLzEATY5n0QLvkzB5woXuxYL5
GQOHMnSxNpLmA3oMcUXuWAnYCZO36vca6Q==
-----END EC PRIVATE KEY-----`,
		},
		admin: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICdzCCAhygAwIBAgIDCflpMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABLvmllDVbuh2
GtbuCljm30QahunesOn4b2bguYSUlAvPtPQJYXgPumvjdzoJNLEuxc1/4APtenVV
W50oHOw1GOCjajBoMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQgIChdYs97+Kol
bT+ejoZ2nSoRi96Y29u1TAm/83hG22UwKwYDVR0jBCQwIoAg7B1IxX3SGJWvunbV
Fthcz+f6SWpvn5gtTvSeS1AU400wCgYIKoZIzj0EAwIDSQAwRgIhAKe385uRkeVx
7Y2YHBcaw4Fq8hoJvw5KTvU1WXzts4ftAiEAtwIIN1e9c6IZSO9HYGxCS2tn38KC
wgo3Na9mff4Eb18=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICHQ+2rVlPUYzwDgjrPitbMqWHgEdCKWJUOcXH3AALO0oAoGCCqGSM49
AwEHoUQDQgAEu+aWUNVu6HYa1u4KWObfRBqG6d6w6fhvZuC5hJSUC8+09AlheA+6
a+N3Ogk0sS7FzX/gA+16dVVbnSgc7DUY4A==
-----END EC PRIVATE KEY-----`,
		},
		client: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICeTCCAh6gAwIBAgIDCbiOMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmcx
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEKwPtxQNg
iXFO9braR8vb3zsoYPUMPFJOgpELyc7x5Ui53VGyM1vZ9kepiTLjsvKmQ1NNw1LA
Izqdy+ImJfLrHKNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCDFrrdkDk7Q
qDyHUbqfsYlFyxrjEBZ9qBQ8VnWErXI59DArBgNVHSMEJDAigCDsHUjFfdIYla+6
dtUW2FzP5/pJam+fmC1O9J5LUBTjTTAKBggqhkjOPQQDAgNJADBGAiEAiR6A+9wu
aQjlx5eSGY+Y26RqbyQuPUsJ+lYEWdocqtoCIQC5Ma91+EpgnvKVKk7PMs8R3x0d
Y/+mm9RigURoqOqmfQ==
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILEdLM3wtfLx8VsgKQZEAqdGoUBnE45GRuCT6QU0wa5CoAoGCCqGSM49
AwEHoUQDQgAEKwPtxQNgiXFO9braR8vb3zsoYPUMPFJOgpELyc7x5Ui53VGyM1vZ
9kepiTLjsvKmQ1NNw1LAIzqdy+ImJfLrHA==
-----END EC PRIVATE KEY-----`,
		},
	},
	org2Name: {
		orgId: org2Name,
		consensusNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICsTCCAlegAwIBAgIDCsRTMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmcyLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
8EWDAFdUmUhHQgps5XUB/h9atcv27luB1JuWQXz6ny+sC8FOeVfE9rO9zKQYFWO3
iCoh4GrYxRdthm4eenQqlaOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIE
ID7tMZinxkE5FidQ1MYlpNdhkSPl9Q6Io64HUn93ltPbMCsGA1UdIwQkMCKAIBGu
8Qs7ND6yIKHIL/1r3DwkDv7PeZMjpP0PbjIu7fDTMC8GC4EnWI9kCx6PZAsEBCBk
NzIzZTVhYWZiZjU0MWVhODYxNDBmMWNhZWFiZTMzMzAKBggqhkjOPQQDAgNIADBF
AiEAyZmC8w2HW0OBLoadJruJ0SEcR/dozXkamVk/aHZaMm0CIFai1uNy0MpRHDlz
FHQQIm/C3zNVeqEeQtj8D99xO1Ub
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAvm9UjLpLcB8wjkDR9HDImj3cPDDHRMgaT0leIeR4TeoAoGCCqGSM49
AwEHoUQDQgAE8EWDAFdUmUhHQgps5XUB/h9atcv27luB1JuWQXz6ny+sC8FOeVfE
9rO9zKQYFWO3iCoh4GrYxRdthm4eenQqlQ==
-----END EC PRIVATE KEY-----`,
		},
		commonNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICrDCCAlGgAwIBAgIDCoPPMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcy
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEGjiyxkcU
k5CQjhr25Q5CAiQ2SFcpwkASPvYj1LYGuWquXldMoTY5++tGwTOlkaZpDLfcT3xT
FYoCkblbXs9HAqOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIEILtGgXK3
XwEp8sHv0ky9UiVyfaQ9uWCIs2sdACGuBVSxMCsGA1UdIwQkMCKAIBGu8Qs7ND6y
IKHIL/1r3DwkDv7PeZMjpP0PbjIu7fDTMC8GC4EnWI9kCx6PZAsEBCBjODc2NmIw
NGY5ZWY0YWU5YWI1MTgzYjIyN2MwY2Y0YTAKBggqhkjOPQQDAgNJADBGAiEAkIQf
qGYaaoPL4AJp+JZU8MvoJneuoApVOZuljkEbNlwCIQDEWaNkzKkr0AqJ1M3DdupH
yGmWgBCM+UxZCqMQBZFOHQ==
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIATXmaJ9ZWcCJEr/6gR/IzUw7M7w314aJtnsgsC3djwhoAoGCCqGSM49
AwEHoUQDQgAEGjiyxkcUk5CQjhr25Q5CAiQ2SFcpwkASPvYj1LYGuWquXldMoTY5
++tGwTOlkaZpDLfcT3xTFYoCkblbXs9HAg==
-----END EC PRIVATE KEY-----`,
		},
		admin: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICdzCCAhygAwIBAgIDCh8vMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMi5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABEnOz612uwoz
7OeKPJR75qSRDI3eaVxXvexK+TC2/tx0/oFb0cz9L+LFHA2EoUYTCsV7khlJLfri
W10I+pOnmM2jajBoMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQgQVxi8JdvsAxL
mroqIgLng115zEV1unZP/pdIRB3VIpgwKwYDVR0jBCQwIoAgEa7xCzs0PrIgocgv
/WvcPCQO/s95kyOk/Q9uMi7t8NMwCgYIKoZIzj0EAwIDSQAwRgIhAM2Bm+MBh7IC
VfQRIadCIJZRNJ5sdPmH9idVpp4kZL1KAiEAslHlto1sGCCSvTN7Yh/LjEtWsfEJ
PqDg+9VaWSWwRPc=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKGNP8bB/MbsR9k0onfKz4MF8lY8vZHM16yK6QTc8sVWoAoGCCqGSM49
AwEHoUQDQgAESc7PrXa7CjPs54o8lHvmpJEMjd5pXFe97Er5MLb+3HT+gVvRzP0v
4sUcDYShRhMKxXuSGUkt+uJbXQj6k6eYzQ==
-----END EC PRIVATE KEY-----`,
		},
		client: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDAXzUMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmcy
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE2U5A6TSC
fhXM28eoK15b6ECWzgyBi76uNDSWeXGo1xnJ95twcoypsi5CM/WHc35+V/1M/9Dd
/HUWM1v18IYwTKNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCDEijhdRrac
J4rCkW1Q3RkmvrHbuDdrNPzRaPW9oQ0F3jArBgNVHSMEJDAigCARrvELOzQ+siCh
yC/9a9w8JA7+z3mTI6T9D24yLu3w0zAKBggqhkjOPQQDAgNIADBFAiAtd1UXWsfP
FIro7w6NK8Mn8lEHwzyCHryuB6DYy0HKegIhAMNmPIVBHERvbjuqDMyaayhVJLL/
9ImD5XhOknoYNXDu
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIDUsYYf2Rr7sLiTGxBq963VnrusQuvQrSIQib3cpwY80oAoGCCqGSM49
AwEHoUQDQgAE2U5A6TSCfhXM28eoK15b6ECWzgyBi76uNDSWeXGo1xnJ95twcoyp
si5CM/WHc35+V/1M/9Dd/HUWM1v18IYwTA==
-----END EC PRIVATE KEY-----`,
		},
	},
	org3Name: {
		orgId: org3Name,
		consensusNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICsDCCAlegAwIBAgIDCRfJMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmczLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
0TLhhf3wD8Tt7gQgYUFEuohpEt2kGuBxplZCnOBbSyZAg36yiAKYGKrN9MUi/Gqr
CdxZORGKU4kP4zwnPxrUKqOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIE
IMFQnc4jmJG19LQj5fSeN5Bs/K2tkLH86oOArTJ02u/cMCsGA1UdIwQkMCKAIPiR
rieswMPIUQdqDG0flHnnfXrEx7wmyMnX1E+sXQmXMC8GC4EnWI9kCx6PZAsEBCAz
YWEwYWFmOTAyZTU0NjA0ODk0NzQzNTU5NTg5MTA3YTAKBggqhkjOPQQDAgNHADBE
AiBzTBxxLoXwfUqrv4/jDIC13jjjwt9BR7uE9EFj+wMxtQIgRA1Ork13rmk9iHO0
v6e5P2tYZSzoQeZN0AEOEhqg/yw=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIE91kRaEt1SQX+/g/NB4Nqm0v3+B1pCNFjqOGrzTgXCQoAoGCCqGSM49
AwEHoUQDQgAE0TLhhf3wD8Tt7gQgYUFEuohpEt2kGuBxplZCnOBbSyZAg36yiAKY
GKrN9MUi/GqrCdxZORGKU4kP4zwnPxrUKg==
-----END EC PRIVATE KEY-----`,
		},
		commonNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICqzCCAlGgAwIBAgIDAaGRMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmcz
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE1M7ulV6r
F3xpVWNjUTrAzzPTyRtpUD+mavHjy7DOg3nAMrN58KdDkIJjZG8QmgBBz4rrEd6p
CtvBe8S9B2oYqaOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIEIDydh+AP
I2FS5Z26fCTHoon8gl2poEqgslDyaXF/Ac7sMCsGA1UdIwQkMCKAIPiRrieswMPI
UQdqDG0flHnnfXrEx7wmyMnX1E+sXQmXMC8GC4EnWI9kCx6PZAsEBCAyNDU2ODQ4
MjY1NDM0ZDgzOWRmODRiZDhhMmM4ODg0YzAKBggqhkjOPQQDAgNIADBFAiA5qfxi
dz8FJWH0knukYxtUnSaVnHZPxjBm8BL+xWI+KAIhAL9Bv324dLvxrMJcuE7DrWFT
tbjAS9wWZR2lS2DRUV2Z
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKNwF8ztZFnprp75qEEpxVRkdeT/wBgDaGu9YYjZOrlToAoGCCqGSM49
AwEHoUQDQgAE1M7ulV6rF3xpVWNjUTrAzzPTyRtpUD+mavHjy7DOg3nAMrN58KdD
kIJjZG8QmgBBz4rrEd6pCtvBe8S9B2oYqQ==
-----END EC PRIVATE KEY-----`,
		},
		admin: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICdzCCAhygAwIBAgIDBOyxMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMy5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABA3l7OrMJBBl
dtu8G+rvsOliQ+TovFm1KeENxkgzHZtGuScUdwxx/XvVRp/h/SlCD9ZBWWjCPwtG
2CjqbDlOkf+jajBoMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQgMgyqbDMOen/+
aywHVRsZJFtm51znx28sMo36HYFqYb8wKwYDVR0jBCQwIoAg+JGuJ6zAw8hRB2oM
bR+Ueed9esTHvCbIydfUT6xdCZcwCgYIKoZIzj0EAwIDSQAwRgIhAOVcKFhjMnvL
o/HqvacCRahdHRPVETYgIW0RDbWELnRJAiEAo490ry4tQixZsT1tunMCtfSqv3/P
n73HOZKWVJOpiDY=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEII0ArhPe4yH/BXXqb2CgvIx1OBoO9EcqReupgPafswVDoAoGCCqGSM49
AwEHoUQDQgAEDeXs6swkEGV227wb6u+w6WJD5Oi8WbUp4Q3GSDMdm0a5JxR3DHH9
e9VGn+H9KUIP1kFZaMI/C0bYKOpsOU6R/w==
-----END EC PRIVATE KEY-----`,
		},
		client: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICeTCCAh6gAwIBAgIDB314MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmcz
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAExL1ynO/I
bLGzM7bXfA8y8q5fECai2iy0GAZhKQ52EKxFSqYVm02uIYMtvVcFSrNcXMi4kA4g
2EsXRtdafu06WqNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCCGbTjv1WlZ
fBKKVEE+jl+tqVp9bbQexuCMa/tJeG90EjArBgNVHSMEJDAigCD4ka4nrMDDyFEH
agxtH5R55316xMe8JsjJ19RPrF0JlzAKBggqhkjOPQQDAgNJADBGAiEA0P6logfL
CNZFW5EjgN+HbAiehZFLLApkkUa3WQWdF3QCIQCETCQA7GXgZFlAAA5plUEZYg56
UIuiy77TJ0nSqvBZYg==
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILCJ8WWebTs3UyZqH9VW4gpBSIzng9nPnCZ1pOhe8aZxoAoGCCqGSM49
AwEHoUQDQgAExL1ynO/IbLGzM7bXfA8y8q5fECai2iy0GAZhKQ52EKxFSqYVm02u
IYMtvVcFSrNcXMi4kA4g2EsXRtdafu06Wg==
-----END EC PRIVATE KEY-----`,
		},
	},
	org4Name: {
		orgId: org4Name,
		consensusNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICsDCCAlagAwIBAgICVewwCgYIKoZIzj0EAwIwgYoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1v
cmc0LmNoYWlubWFrZXIub3JnMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMT
GWNhLnd4LW9yZzQuY2hhaW5tYWtlci5vcmcwHhcNMjIwMTA1MDg0MTM3WhcNMzIw
MTAzMDg0MTM3WjCBlzELMAkGA1UEBhMCQ04xEDAOBgNVBAgTB0JlaWppbmcxEDAO
BgNVBAcTB0JlaWppbmcxHzAdBgNVBAoTFnd4LW9yZzQuY2hhaW5tYWtlci5vcmcx
EjAQBgNVBAsTCWNvbnNlbnN1czEvMC0GA1UEAxMmY29uc2Vuc3VzMS5zaWduLnd4
LW9yZzQuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASX
9x1eDWWxM1RK5VRbZwvQVo2EgZ44UT/LqXPYLnfjjck1Fkk2lOnTXZZXmkYNWhf+
2QYxRTOhdkQoP5UAY/nso4GcMIGZMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQg
frmlHnvL8D2zM76+Dz1TTjBgqwyexTXqqxLOb4NI37AwKwYDVR0jBCQwIoAg1Y39
mnOI62mH7yJ8rWYArk2Uu54QQjOc8wx/0gjlZccwLwYLgSdYj2QLHo9kCwQEIDQy
M2Y4OWQxZWYyYjRmZTk5NjU5Y2M5ZDIyM2Q3ZGI4MAoGCCqGSM49BAMCA0gAMEUC
IGLFid8qnPHxGCPKMh2gjOohE2QZdnpSCBDrn9ZjquOdAiEAi5ENBuk8i0ER5vEj
6c2mzhAeYeFlajhgrR/OqGDBtnQ=
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIDYIT83M/43JygDzONED38qHVF7kIe0WpP47FmbxntqfoAoGCCqGSM49
AwEHoUQDQgAEl/cdXg1lsTNUSuVUW2cL0FaNhIGeOFE/y6lz2C53443JNRZJNpTp
012WV5pGDVoX/tkGMUUzoXZEKD+VAGP57A==
-----END EC PRIVATE KEY-----`,
		},
		commonNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICqzCCAlGgAwIBAgIDA7DEMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc0LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc0LmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmc0
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE4F6Oa3OI
FWi6ZbbhrCjXWKPq77Is3bh6yyM/zQLYyRqV8gVl1qNWY2BdiXvP1zcOniOEJL06
jLNqNiBN26Ae4qOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIEIHqM29Eu
/OIRNSLb2VUHqlTgczsomm2pxU8qR2OCbnzRMCsGA1UdIwQkMCKAINWN/ZpziOtp
h+8ifK1mAK5NlLueEEIznPMMf9II5WXHMC8GC4EnWI9kCx6PZAsEBCBlZTk0NGFi
MDgwN2M0NWJlOWFmZmU1MTM1NjQwNTMzOTAKBggqhkjOPQQDAgNIADBFAiAMELs7
dn5tEz0klVEOGfdO2F7JZB3xGt6j7N9Yh5UWlgIhALt6jDZ0r1FHD/3ahMJ0lu7H
dR9HvxXYH4EIDov6n0PK
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKSOA9iCEAhPWJL04BqDVvPuN+8UhcHIoaXSuaDwAZpCoAoGCCqGSM49
AwEHoUQDQgAE4F6Oa3OIFWi6ZbbhrCjXWKPq77Is3bh6yyM/zQLYyRqV8gVl1qNW
Y2BdiXvP1zcOniOEJL06jLNqNiBN26Ae4g==
-----END EC PRIVATE KEY-----`,
		},
		admin: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICdjCCAhygAwIBAgIDCWOWMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc0LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc0LmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnNC5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABIpdgLUion6c
FNpxW6KP4AiPnAJSVwfiI4ZGKUspLC+URyBskgKjW3C3d8MtUewPn8jvtkekpgEN
MY8txGaEue2jajBoMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQglnuCCQsINAwt
WRYlPuDQKjAvDAYZuNCmZni6eA3yIAIwKwYDVR0jBCQwIoAg1Y39mnOI62mH7yJ8
rWYArk2Uu54QQjOc8wx/0gjlZccwCgYIKoZIzj0EAwIDSAAwRQIhAIs+w5nqMCOV
pHNaxnQYl4oQEjWShQpW75Ymig8lQYyeAiAsySvYYysAYqwTwd/dRhO+xXcmih57
7PgEXqH50uHo8g==
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIHAmxIY3rg80XXtpl8pJcWGyoNtvBvnTHfsadWgBddLxoAoGCCqGSM49
AwEHoUQDQgAEil2AtSKifpwU2nFboo/gCI+cAlJXB+IjhkYpSyksL5RHIGySAqNb
cLd3wy1R7A+fyO+2R6SmAQ0xjy3EZoS57Q==
-----END EC PRIVATE KEY-----`,
		},
		client: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICeDCCAh6gAwIBAgIDBfdBMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNC5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc0LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc0LmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmc0
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn2/BmgVH
2umLZB+FIIQoSFnAQIXaNDIGZheexMOuwmbzat3IryCU/cdkzWdcAYeBHzWOsb66
Dw/32vM9F+kwvKNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCBOL1EUVdvD
VMnxkPnfqrd7NIW1yCLaEubD51biTT0mfzArBgNVHSMEJDAigCDVjf2ac4jraYfv
InytZgCuTZS7nhBCM5zzDH/SCOVlxzAKBggqhkjOPQQDAgNIADBFAiEAwO3TaZ6+
a7qBqahRN9NMpiQ564lcthogX/NC5veHoiACIHrPbVfmfJZrsWi8XQYdWmGLMQQm
/2ek6gv0z+omVIpu
-----END CERTIFICATE-----`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIOiF9f8Fm7wIN/pX7gXNVQcIlO4tENXbIQzvbtZm8OXaoAoGCCqGSM49
AwEHoUQDQgAEn2/BmgVH2umLZB+FIIQoSFnAQIXaNDIGZheexMOuwmbzat3IryCU
/cdkzWdcAYeBHzWOsb66Dw/32vM9F+kwvA==
-----END EC PRIVATE KEY-----`,
		},
	},
	org5Name: {
		orgId: org5Name,
		consensusNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICsTCCAlegAwIBAgIDCWTrMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZcxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwljb25zZW5zdXMxLzAtBgNVBAMTJmNvbnNlbnN1czEuc2lnbi53
eC1vcmc1LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
oA8GzR+yDoKNSWoK1HrdHhf3t+uR/Jl/TWldM+vOltEHUh6ntahQt6i/2cKGvopM
dF3QOYV9NTDM1Qa4wchbiaOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIE
IF0Q5lc9L8nCfwx9b3LkOo3H5MhkOB0PIWGbf1YQ9Ts5MCsGA1UdIwQkMCKAIC+o
bFpuFlB3XapWTXSNTDj7omThfiFvAQjCGHSUiXmIMC8GC4EnWI9kCx6PZAsEBCBh
OTYwZjcxY2NlMTU0NmJjYmQ5Nzc5MjYyNmFjZGQ0NjAKBggqhkjOPQQDAgNIADBF
AiEAuntppGKvoN9ZSUT4ZUYHVcoGRID81gzkRlJH+reFgV8CIGnPXmpccDc9d0bI
Lxlk8r5q7/RxU74CuD2L4igjXka6
-----END CERTIFICATE-----
`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIBms6aedYHR3oEfYLqqeYmORkuzNkVuhDCoAdANu3uKwoAoGCCqGSM49
AwEHoUQDQgAEoA8GzR+yDoKNSWoK1HrdHhf3t+uR/Jl/TWldM+vOltEHUh6ntahQ
t6i/2cKGvopMdF3QOYV9NTDM1Qa4wchbiQ==
-----END EC PRIVATE KEY-----
`,
		},
		commonNode: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICrDCCAlGgAwIBAgIDDGEQMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjb21tb24xLDAqBgNVBAMTI2NvbW1vbjEuc2lnbi53eC1vcmc1
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEtkzyUfB7
t3hyjyzfQ06YcngARLDyaxTEtuaroOxrN2+e+GvAGv6cB2VK0kSTAlThi1ATycPV
lxAKtrqMC22yVaOBnDCBmTAOBgNVHQ8BAf8EBAMCBsAwKQYDVR0OBCIEIActif1/
XaM71aPqPhotcFjCDx70/I8SgMN+1+2BrmNbMCsGA1UdIwQkMCKAIC+obFpuFlB3
XapWTXSNTDj7omThfiFvAQjCGHSUiXmIMC8GC4EnWI9kCx6PZAsEBCA5YjUzYWI4
NjI2N2I0NWZlODEyMGU2ZjY5YjE0MzlkMTAKBggqhkjOPQQDAgNJADBGAiEAqYB9
O+GcIzXjWEYMbcObo39JA0kzBuVj98HKTj234aMCIQDBRBJHFpH9/WnbLPyv0wqQ
y8Ex44dIpCtfsdpQ/szpFw==
-----END CERTIFICATE-----
`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIK2vIApIjWFArwXhniIEyjQ+xVQPdqzcT9glofVH97BYoAoGCCqGSM49
AwEHoUQDQgAEtkzyUfB7t3hyjyzfQ06YcngARLDyaxTEtuaroOxrN2+e+GvAGv6c
B2VK0kSTAlThi1ATycPVlxAKtrqMC22yVQ==
-----END EC PRIVATE KEY-----
`,
		},
		admin: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICdTCCAhygAwIBAgIDAtpDMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnNS5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABNc84O0bCBRf
JyZtnndSKW79ZXI2yHuDQwIzWrPqn2egR76ii4mm8F2DzwnrClb0Q6XFh0DiAKKG
W68dqln0u1SjajBoMA4GA1UdDwEB/wQEAwIGwDApBgNVHQ4EIgQgrQc+KJ7wN0bw
x7GUWHCb8osjneWMUv+UMtr6GhSlSg4wKwYDVR0jBCQwIoAgL6hsWm4WUHddqlZN
dI1MOPuiZOF+IW8BCMIYdJSJeYgwCgYIKoZIzj0EAwIDRwAwRAIgJIOjbdySqJt1
bqCuym7eH67w+w4CliPjCZrdrUWTq/8CIFzS8MtH7VS+wjsLEribUPTZge+bwC/4
249wC9ZH7A2H
-----END CERTIFICATE-----
`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILVR0LzYKfSb4GxjXoFOf5g+6UdKgovHCHGJxiGRvmo5oAoGCCqGSM49
AwEHoUQDQgAE1zzg7RsIFF8nJm2ed1Ipbv1lcjbIe4NDAjNas+qfZ6BHvqKLiabw
XYPPCesKVvRDpcWHQOIAooZbrx2qWfS7VA==
-----END EC PRIVATE KEY-----
`,
		},
		client: certificatePair{
			certificate: `-----BEGIN CERTIFICATE-----
MIICeTCCAh6gAwIBAgIDCLkBMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIyMDEwNTA4NDEzN1oXDTMy
MDEwMzA4NDEzN1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmc1
LmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEHjXunDU8
mNDU2vfwuCP++Al1nit824+AvxFI1zm2gFDazZAH2Uo7ItSURM6Ft+vlTlGh9EJ+
TDsSbEIJ2Sb5KaNqMGgwDgYDVR0PAQH/BAQDAgbAMCkGA1UdDgQiBCBgqiUYjfDP
sYO9f+kKADUSR+rojorhgNKOybUdagjAiDArBgNVHSMEJDAigCAvqGxabhZQd12q
Vk10jUw4+6Jk4X4hbwEIwhh0lIl5iDAKBggqhkjOPQQDAgNJADBGAiEA8rdRTP4i
L9ndIyc1e+oFF0/lw2g61v8ti2Hvuk66mqsCIQDlGnh5PqfF8/D0Ecm4/P2Gppk+
inIRftl5SkBxaqSLGQ==
-----END CERTIFICATE-----
`,
			sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINdppULgY5PvQ21Hxz02rZ2/KUYEHrTPmz8C6zLepPePoAoGCCqGSM49
AwEHoUQDQgAEHjXunDU8mNDU2vfwuCP++Al1nit824+AvxFI1zm2gFDazZAH2Uo7
ItSURM6Ft+vlTlGh9EJ+TDsSbEIJ2Sb5KQ==
-----END EC PRIVATE KEY-----
`,
		},
	},
}

var acsMap = map[string]ac{}

func createTempDirWithCleanFunc() (string, func(), error) {
	var td = filepath.Join("./temp")
	err := os.MkdirAll(td, os.ModePerm)
	if err != nil {
		return "", nil, err
	}
	var cleanFunc = func() {
		_ = os.RemoveAll(td)
		_ = os.RemoveAll(filepath.Join("./default.log"))
		now := time.Now()
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
		now = now.Add(-5 * time.Hour)
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
	}
	return td, cleanFunc, nil
}

func constructAC(t *testing.T, info orgInfo) ac {
	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, info.orgId+".key")
	localCertFile := filepath.Join(td, info.orgId+".crt")
	err = ioutil.WriteFile(localPrivKeyFile, []byte(info.consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(info.consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, info.orgId, nil, logger)
	require.Nil(t, err)
	consensusMember, err := acInst.NewMemberFromCertPem(info.orgId, info.consensusNode.certificate)
	require.Nil(t, err)
	consensusNode, err := acInst.NewSigningMember(consensusMember, info.consensusNode.sk, "")
	require.Nil(t, err)
	commonMember, err := acInst.NewMemberFromCertPem(info.orgId, info.commonNode.certificate)
	require.Nil(t, err)
	commonNode, err := acInst.NewSigningMember(commonMember, info.commonNode.sk, "")
	require.Nil(t, err)
	adminMember, err := acInst.NewMemberFromCertPem(info.orgId, info.admin.certificate)
	require.Nil(t, err)
	admin, err := acInst.NewSigningMember(adminMember, info.admin.sk, "")
	require.Nil(t, err)
	clientMember, err := acInst.NewMemberFromCertPem(info.orgId, info.client.certificate)
	require.Nil(t, err)
	client, err := acInst.NewSigningMember(clientMember, info.client.sk, "")
	require.Nil(t, err)
	return ac{
		acInst:        acInst,
		consensusNode: consensusNode,
		commonNode:    commonNode,
		admin:         admin,
		client:        client,
	}
}

func TestNewAccessControlWithChainConfig(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
}

func TestAccessControlGetHashAlg(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	fmt.Printf("hash alg: %s\n", acInst.GetHashAlg())
}

func TestAccessControlValidateResourcePolicy(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	resourcePolicy := &config.ResourcePolicy{
		ResourceName: "INIT_CONTRACT",
		Policy:       &pbac.Policy{Rule: "ANY"},
	}
	ok := acInst.ValidateResourcePolicy(resourcePolicy)
	require.Equal(t, true, ok)
	resourcePolicy = &config.ResourcePolicy{
		ResourceName: "P2P",
		Policy:       &pbac.Policy{Rule: "ANY"},
	}
	ok = acInst.ValidateResourcePolicy(resourcePolicy)
	require.Equal(t, false, ok)
}

func TestAccessControlLookUpResourceIdByTxType(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	resourceId, err := acInst.LookUpResourceNameByTxType(common.TxType_INVOKE_USER_CONTRACT)
	require.Nil(t, err)
	require.Equal(t, protocol.ResourceNameWriteData, resourceId)

	_, err = acInst.LookUpResourceNameByTxType(common.TxType(888))
	require.NotNil(t, err)
}

func TestAccessControlGetLocalOrgId(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	require.Equal(t, acInst.GetLocalOrgId(), org1Name)
}

func TestAccessControlGetLocalSigningMember(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	signingMember := acInst.GetLocalSigningMember()
	require.NotNil(t, signingMember)
	_, err = signingMember.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
}

func TestAccessControlNewMemberFromCertPem(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
}

func TestAccessControlNewMemberFromProto(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	signingMember := acInst.GetLocalSigningMember()
	require.NotNil(t, signingMember)
	signerRead, err := signingMember.GetSerializedMember(true)
	signer, err := acInst.NewMemberFromProto(signerRead)
	require.Nil(t, err)
	require.NotNil(t, signer)
}

func TestAccessControlNewSigningMemberFromCertFile(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	signer, err := acInst.NewSigningMemberFromCertFile(org1Name, localPrivKeyFile, "", localCertFile)
	require.Nil(t, err)
	require.NotNil(t, signer)
}

func TestAccessControlNewSigningMember(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	signer, err := acInst.NewSigningMember(member, orgList[org1Name].consensusNode.sk, "")
	require.Nil(t, err)
	require.NotNil(t, signer)
}

func TestAccessControlCreatePrincipalAndGetValidEndorsementsAndVerifyPrincipal(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	for orgId, info := range orgList {
		acsMap[orgId] = constructAC(t, info)
	}
	acInst := acsMap[org1Name].acInst

	// read
	sigRead, err := acsMap[org1Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err := acsMap[org1Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementReadZephyrus := &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err := acInst.CreatePrincipal(protocol.ResourceNameReadData, []*common.EndorsementEntry{endorsementReadZephyrus}, []byte(msg))
	require.Nil(t, err)
	ok, err := acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	commonNodeSigner, err := acsMap[org1Name].acInst.NewMemberFromProto(signerRead)
	require.Nil(t, err)
	commonNodeSignerCached, err := acsMap[org2Name].acInst.NewMemberFromProto(signerRead)
	require.Nil(t, err)
	commonNodeSignerBytes, err := commonNodeSigner.Serialize(true)
	require.Nil(t, err)
	commonNodeSignerCachedBytes, err := commonNodeSignerCached.Serialize(true)
	require.Nil(t, err)
	require.Equal(t, string(commonNodeSignerBytes), string(commonNodeSignerCachedBytes))
	validEndorsements, err := acsMap[org2Name].acInst.GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 1)
	require.Equal(t, endorsementReadZephyrus.String(), validEndorsements[0].String())
	// read invalid
	sigRead, err = acsMap[org5Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org5Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead := &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameReadData, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// wrong signer
	sigRead, err = acsMap[org5Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org2Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameReadData, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// write
	sigRead, err = acsMap[org1Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameWriteData, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	sigRead, err = acsMap[org1Name].client.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].client.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameTxTransact, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	sigRead, err = acsMap[org1Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameTxTransact, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// P2P
	sigRead, err = acsMap[org1Name].consensusNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].consensusNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	sigRead, err = acsMap[org4Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org4Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org3Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	sigRead, err = acsMap[org1Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// consensus
	sigRead, err = acsMap[org1Name].consensusNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org1Name].consensusNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	sigRead, err = acsMap[org4Name].commonNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org4Name].commonNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org3Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// self
	sigRead, err = acsMap[org4Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org4Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementRead}, []byte(msg), org4Name)
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_TRUST_ROOT_UPDATE.String(), []*common.EndorsementEntry{endorsementRead}, []byte(msg), org4Name, acsMap[org2Name].acInst)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	sigRead, err = acsMap[org3Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerRead, err = acsMap[org3Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementRead = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = acInst.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementRead}, []byte(msg), org4Name)
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_TRUST_ROOT_UPDATE.String(), []*common.EndorsementEntry{endorsementRead}, []byte(msg), org4Name, acsMap[org2Name].acInst)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// majority
	sigEurus, err := acsMap[org4Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerEurus, err := acsMap[org4Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementEurus := &common.EndorsementEntry{
		Signer:    signerEurus,
		Signature: sigEurus,
	}
	sigAuster, err := acsMap[org3Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerAuster, err := acsMap[org3Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementAuster := &common.EndorsementEntry{
		Signer:    signerAuster,
		Signature: sigAuster,
	}
	sigZephyrus, err := acsMap[org1Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerZephyrus, err := acsMap[org1Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementZephyrus := &common.EndorsementEntry{
		Signer:    signerZephyrus,
		Signature: sigZephyrus,
	}
	sigBoreas, err := acsMap[org2Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerBoreas, err := acsMap[org2Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementBoreas := &common.EndorsementEntry{
		Signer:    signerBoreas,
		Signature: sigBoreas,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err = acsMap[org2Name].acInst.GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)
	principalRead, err = acInst.CreatePrincipal(common.ConfigFunction_CONSENSUS_EXT_ADD.String(), []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err = acsMap[org2Name].acInst.GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)
	require.Equal(t, endorsementAuster.String(), validEndorsements[0].String())
	require.Equal(t, endorsementBoreas.String(), validEndorsements[1].String())
	require.Equal(t, endorsementZephyrus.String(), validEndorsements[2].String())
	ok, err = utils.VerifyConfigUpdateTx(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus}, []byte(msg), "", acsMap[org2Name].acInst)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// abnormal
	sigThuellai, err := acsMap[org5Name].admin.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerThuellai, err := acsMap[org5Name].admin.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementThuellai := &common.EndorsementEntry{
		Signer:    signerThuellai,
		Signature: sigThuellai,
	}
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err = acsMap[org2Name].acInst.GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_CORE_UPDATE.String(), []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementZephyrus, endorsementEurus}, []byte(msg), "", acsMap[org2Name].acInst)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementAuster}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	validEndorsements, err = acsMap[org2Name].acInst.GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_CORE_UPDATE.String(), []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementAuster}, []byte(msg), "", acsMap[org2Name].acInst)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// all
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// abnormal
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus, endorsementThuellai}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// mock sign
	endorsements, err := MockSignWithMultipleNodes([]byte(msg), []protocol.SigningMember{acsMap[org1Name].admin, acsMap[org2Name].admin, acsMap[org4Name].admin}, acInst.GetHashAlg())
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_CORE_UPDATE.String(), endorsements, []byte(msg), "", acsMap[org2Name].acInst)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	endorsements, err = MockSignWithMultipleNodes([]byte(msg), []protocol.SigningMember{acsMap[org2Name].admin, acsMap[org4Name].admin}, acInst.GetHashAlg())
	ok, err = utils.VerifyConfigUpdateTx(common.ConfigFunction_CORE_UPDATE.String(), endorsements, []byte(msg), "", acsMap[org2Name].acInst)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// threshold
	policyLimit, err := acInst.CreatePrincipal("test_2", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyLimit)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	policyLimit, err = acInst.CreatePrincipal("test_2_admin", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyLimit)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	policyLimit, err = acInst.CreatePrincipal("test_2", []*common.EndorsementEntry{endorsementAuster, endorsementThuellai}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyLimit)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	policyLimit, err = acInst.CreatePrincipal("test_2_admin", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyLimit)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// portion
	policyPortion, err := acInst.CreatePrincipal("test_3/4", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyPortion)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	policyPortion, err = acInst.CreatePrincipal("test_3/4_admin", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyPortion)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	policyPortion, err = acInst.CreatePrincipal("test_3/4", []*common.EndorsementEntry{endorsementAuster, endorsementAuster, endorsementBoreas, endorsementThuellai}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyPortion)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	policyPortion, err = acInst.CreatePrincipal("test_3/4_admin", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus, endorsementEurus}, []byte(msg))
	require.Nil(t, err)
	ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyPortion)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// bench
	var timeStart, timeEnd int64
	count := int64(100)
	// any
	principalRead, err = acInst.CreatePrincipal(protocol.ResourceNameReadData, []*common.EndorsementEntry{endorsementRead}, []byte(msg))
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify ANY average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
	// self
	principalRead, err = acInst.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementEurus}, []byte(msg), org4Name)
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = acsMap[org2Name].acInst.VerifyPrincipal(principalRead)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify SELF average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
	// consensus
	sigZephyrusConsensus, err := acsMap[org1Name].consensusNode.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	signerZephyrusConsensus, err := acsMap[org1Name].consensusNode.GetSerializedMember(true)
	require.Nil(t, err)
	endorsementZephyrusConsensus := &common.EndorsementEntry{
		Signer:    signerZephyrusConsensus,
		Signature: sigZephyrusConsensus,
	}
	policyConsensus, err := acInst.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementZephyrusConsensus}, []byte(msg))
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = acsMap[org2Name].acInst.VerifyPrincipal(policyConsensus)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify CONSENSUS average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
}

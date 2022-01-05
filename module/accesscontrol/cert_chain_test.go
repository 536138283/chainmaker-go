/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	rawChainTemplate    = "raw chain: %v\n"
	sortedChainTemplate = "sorted chain: %v\n"
)

var (
	sans = []string{"127.0.0.1", "localhost", "chainmaker.org", "8.8.8.8"}
)

var rootCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICFTCCAbugAwIBAgIIKY2siu2a1YcwCgYIKoZIzj0EAwIwYjELMAkGA1UEBhMC
Q04xEDAOBgNVBAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoT
CG9yZy1yb290MQ0wCwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MB4XDTIyMDEw
NTA5MzcyNVoXDTMyMDEwMzA5MzcyNVowYjELMAkGA1UEBhMCQ04xEDAOBgNVBAgT
B0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoTCG9yZy1yb290MQ0w
CwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEaYKBeDifWTH057RnHaDYdIwfn3T3hF/vYFHiUD4v978BwHDdXT+HKtPn
iN4USd91UITkgOy1ay3Mg/MXVNkDLqNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEIJ2KZDFiQS3Fk07TgzGStJ+gHe7vspve3YdF
TlnB9glMMAsGA1UdEQQEMAKCADAKBggqhkjOPQQDAgNIADBFAiEAxMHwYlF9o4Aw
QSzTYdx/3yWe5ymwYSv06ZAdeABt+w8CIBUSTyECLRd7sMxD3uVt07AYRRXI63rt
FP47nfxGUPKA
-----END CERTIFICATE-----`,
	sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMN5tcoScRTSvaQ2isUzTnkzLKe+ayE2BsNpGj4huByfoAoGCCqGSM49
AwEHoUQDQgAEaYKBeDifWTH057RnHaDYdIwfn3T3hF/vYFHiUD4v978BwHDdXT+H
KtPniN4USd91UITkgOy1ay3Mg/MXVNkDLg==
-----END EC PRIVATE KEY-----`,
}
var intermediateCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICZzCCAgygAwIBAgIIEbHRPyzbGccwCgYIKoZIzj0EAwIwYjELMAkGA1UEBhMC
Q04xEDAOBgNVBAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoT
CG9yZy1yb290MQ0wCwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MB4XDTIyMDEw
NTA5MzcyNVoXDTMyMDEwMzA5MzcyNVowgYMxCzAJBgNVBAYTAkNOMRAwDgYDVQQI
EwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNo
YWlubWFrZXIub3JnMQswCQYDVQQLEwJjYTEiMCAGA1UEAxMZY2Etd3gtb3JnMS5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABNSsYCnJbK9T
E7RmURf4FPYyMFWBs9iMnn01FeXTWto5e2Dl+QGAr15r+4fHE2kiWWrrKJTdV8WC
L+3Jsn25DtKjgYkwgYYwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8w
KQYDVR0OBCIEIHictoBflHUoe+OF9ULTdNYz/XYj2XgmsXiPepaPk6KfMCsGA1Ud
IwQkMCKAIJ2KZDFiQS3Fk07TgzGStJ+gHe7vspve3YdFTlnB9glMMAsGA1UdEQQE
MAKCADAKBggqhkjOPQQDAgNJADBGAiEAtVlfBbOtaZUqBqOelCOhoKlK4E1IUGN3
uUZ+CRcpftMCIQCYwbCagIC6ky28DQponM10IHr5yjVZJccsUwREWpttLQ==
-----END CERTIFICATE-----`,
	sk: `-----BEGIN PRIVATE KEY-----
MHcCAQEEIJc/f+1alDoZcj9xqSLlfga0rtyuJhEH5I2tVV07qiq1oAoGCCqGSM49
AwEHoUQDQgAE1KxgKclsr1MTtGZRF/gU9jIwVYGz2IyefTUV5dNa2jl7YOX5AYCv
Xmv7h8cTaSJZausolN1XxYIv7cmyfbkO0g==
-----END PRIVATE KEY-----`,
}
var leafCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICjjCCAjSgAwIBAgIIAii09pKnXFwwCgYIKoZIzj0EAwIwgYMxCzAJBgNVBAYT
AkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQK
ExZ3eC1vcmcxLmNoYWlubWFrZXIub3JnMQswCQYDVQQLEwJjYTEiMCAGA1UEAxMZ
Y2Etd3gtb3JnMS5jaGFpbm1ha2VyLm9yZzAeFw0yMjAxMDUwOTM4MzRaFw0zMjAx
MDMwOTM4MzRaMHUxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAwDgYD
VQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3JnMQ8w
DQYDVQQLEwZjb21tb24xEDAOBgNVBAMTB2NvbW1vbjEwWTATBgcqhkjOPQIBBggq
hkjOPQMBBwNCAAS82kYBx4kJU7m+UpM49xtylg4rtaXXBseEcbHTU7iZHCFWtOFn
cbG3NMfIkCbKuulAAY/crWpcrtwTL8TmVky7o4GeMIGbMA4GA1UdDwEB/wQEAwID
+DAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUHAwEwKQYDVR0OBCIEICP0YC8o
m2qsDXNYScreygX3+is1vYhol36eBIgQJyFNMCsGA1UdIwQkMCKAIHictoBflHUo
e+OF9ULTdNYz/XYj2XgmsXiPepaPk6KfMBIGA1UdEQQLMAmCB2NvbW1vbjEwCgYI
KoZIzj0EAwIDSAAwRQIhANdeuHcMD5BPsa7OdxLMPm+s1jx3GyXicSt6+RJgzPsI
AiARlURKuaCc5wJNcGojoREtjlJKYxdiua6mWlzTLgqL9w==
-----END CERTIFICATE-----`,
	sk: `-----BEGIN PRIVATE KEY-----
MHcCAQEEIB+0+6Vq/qrrFi0BICjdfrYxLH2BfEhm5hbrS/jq5yyuoAoGCCqGSM49
AwEHoUQDQgAEvNpGAceJCVO5vlKTOPcbcpYOK7Wl1wbHhHGx01O4mRwhVrThZ3Gx
tzTHyJAmyrrpQAGP3K1qXK7cEy/E5lZMuw==
-----END PRIVATE KEY-----`,
}

func TestCertChainFunction(t *testing.T) {
	blockCA, _ := pem.Decode([]byte(rootCert.certificate))
	certRootCA, err := bcx509.ParseCertificate(blockCA.Bytes)
	require.Nil(t, err)
	blockIntermediate, _ := pem.Decode([]byte(intermediateCert.certificate))
	certIntermediate, err := bcx509.ParseCertificate(blockIntermediate.Bytes)
	require.Nil(t, err)
	blockLeaf, _ := pem.Decode([]byte(leafCert.certificate))
	certLeaf, err := bcx509.ParseCertificate(blockLeaf.Bytes)
	require.Nil(t, err)
	rootCertPool := bcx509.NewCertPool()
	rootCertPool.AddCert(certRootCA)
	intermediateCertPool := bcx509.NewCertPool()
	intermediateCertPool.AddCert(certIntermediate)
	chains, err := certIntermediate.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             bcx509.NewCertPool(),
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             intermediateCertPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             bcx509.NewCertPool(),
		Roots:                     intermediateCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	allPool := bcx509.NewCertPool()
	allPool.AddCert(certRootCA)
	allPool.AddCert(certIntermediate)
	allPool.AddCert(certLeaf)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	chains, err = certIntermediate.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	chains, err = certRootCA.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	rootCertAllPool := bcx509.NewCertPool()
	rootCertAllPool.AddCert(certRootCA)
	rootCertAllPool.AddCert(certIntermediate)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertAllPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	rawChain := []*bcx509.Certificate{certIntermediate, certRootCA, certLeaf}
	sortedChain := bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certIntermediate}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certIntermediate, certRootCA}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certRootCA, certIntermediate}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certLeaf}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	fmt.Printf("root: %s, %p\n", hex.EncodeToString(certRootCA.SubjectKeyId), certRootCA)
	fmt.Printf("intermediate: %s, %p\n", hex.EncodeToString(certIntermediate.SubjectKeyId), certIntermediate)
	fmt.Printf("leaf: %s, %p\n", hex.EncodeToString(certLeaf.SubjectKeyId), certLeaf)
}

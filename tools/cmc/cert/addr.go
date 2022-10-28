/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/utils/v2"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	pbconfig "chainmaker.org/chainmaker/pb-go/v2/config"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/spf13/cobra"
)

// certToUserAddrInStake get user addr feature of the DPoS from cert
// @return *cobra.Command
func certToUserAddrInStake() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userAddr",
		Short: "get user addr feature of the DPoS from cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			if len(pubkeyOrCertPath) == 0 {
				return fmt.Errorf("cert or pubkey path is null")
			}
			chainClient, err := sdk.NewChainClient(sdk.WithConfPath(sdkConfPath))
			if err != nil {
				return err
			}

			var (
				authType = chainClient.GetAuthType()
				hashType = chainClient.GetHashType()
			)
			content, err := ioutil.ReadFile(pubkeyOrCertPath)
			if err != nil {
				return fmt.Errorf("read cert content failed, reason: %s", err)
			}

			var pk crypto.PublicKey
			if authType == sdk.PermissionedWithCert {
				if pk, err = getPubkeyFromCert(content); err != nil {
					return err
				}
			} else if authType == sdk.PermissionedWithKey || authType == sdk.Public {
				if pk, err = asym.PublicKeyFromPEM(content); err != nil {
					return err
				}
			}

			addr, err := utils.PkToAddrStr(pk, pbconfig.AddrType_ETHEREUM, hashType)
			if err != nil {
				return fmt.Errorf("pk to addr str failed, reason: %s", err)
			}

			fmt.Printf("address: %s \n\nfrom cert: %s\n", addr, pubkeyOrCertPath)
			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagCertOrPubkeyPath,
	})
	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagCertOrPubkeyPath)
	return cmd
}

// getPubkeyFromCert get public key from a pem format certificate
// @param certContent
// @return []byte
// @return error
func getPubkeyFromCert(certContent []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(certContent)
	if block == nil {
		return nil, errors.New("pem.Decode failed, invalid cert")
	}
	cert, err := bcx509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse cert failed, reason: %s", err)
	}
	return cert.PublicKey, nil
}

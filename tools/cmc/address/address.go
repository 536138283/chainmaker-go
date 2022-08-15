// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package address 关于用户地址的相关命令
package address

import (
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	flagAddressType = "address-type"
	flagHashType    = "hash-type"

	// address types
	addressTypeZXL = "zxl"
	addressTypeCM  = "cm"
	addressTypeEVM = "evm"
)

var (
	addressType string
	hashType    int
	hashAlgoMap = map[int]crypto.HashType{
		0: crypto.HASH_TYPE_SHA256,
		1: crypto.HASH_TYPE_SHA3_256,
		2: crypto.HASH_TYPE_SM3,
	}
)

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&addressType, flagAddressType, "evm", `The type of address obtained.
supported address types zhixinlian: zxl, chainmaker: cm, ethereum: evm 
eg. --address-type=zxl`)

	flags.IntVar(&hashType, flagHashType, 0,
		`The type of hash algo obtained. 0: SAH256 (default), 1: SHA3_256, 2: SM3"
eg. --address-type=0`)
}

// NewAddressCMD new address parse command
func NewAddressCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address",
		Short: "address parse command",
		Long:  "address parse command",
	}

	cmd.AddCommand(newPK2AddrCMD())
	cmd.AddCommand(newHex2AddrCMD())
	cmd.AddCommand(newCert2AddrCMD())

	return cmd
}

// newPK2AddrCMD get address from public key file or pem string
// @return *cobra.Command
func newPK2AddrCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pk-to-addr [public key file path / pem string]",
		Short: "get address from public key file or pem string",
		Long:  "get address from public key file or pem string",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var keyPemStr string
			var isFile = utils.Exists(args[0])
			if isFile {
				keyPem, err := ioutil.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("read key file failed, %s", err)
				}
				keyPemStr = string(keyPem)
			} else {
				keyPemStr = args[0]
			}

			var addr string
			var err error
			hash, ok := hashAlgoMap[hashType]
			if !ok {
				return fmt.Errorf("unsupported hash type %d", hashType)
			}
			switch addressType {
			case addressTypeZXL:
				addr, err = sdk.GetZXAddressFromPKPEM(keyPemStr, hash)
				if err != nil {
					return err
				}
			case addressTypeCM:
				addr, err = sdk.GetCMAddressFromPKPEM(keyPemStr, hash)
				if err != nil {
					return err
				}
			case addressTypeEVM:
				addr, err = sdk.GetEVMAddressFromPKPEM(keyPemStr, hash)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported address type %s", addressType)
			}

			output, err := prettyjson.Marshal(addr)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagAddressType, flagHashType})
	return cmd
}

// newHex2AddrCMD get address from hex string
// @return *cobra.Command
func newHex2AddrCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hex-to-addr [hex string]",
		Short: "get address from hex string",
		Long:  "get address from hex string",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var addr string
			var err error
			hash, ok := hashAlgoMap[hashType]
			if !ok {
				return fmt.Errorf("unsupported hash type %d", hashType)
			}
			switch addressType {
			case addressTypeZXL:
				addr, err = sdk.GetZXAddressFromPKHex(args[0], hash)
				if err != nil {
					return err
				}
			case addressTypeCM:
				addr, err = sdk.GetCMAddressFromPKHex(args[0], hash)
				if err != nil {
					return err
				}
			case addressTypeEVM:
				addr, err = sdk.GetEVMAddressFromPKHex(args[0], hash)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported address type %s", addressType)
			}

			output, err := prettyjson.Marshal(addr)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagAddressType, flagHashType})
	return cmd
}

// newCert2AddrCMD get address from cert file or pem string
// @return *cobra.Command
func newCert2AddrCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert-to-addr [hex string]",
		Short: "get address from cert file or pem string",
		Long:  "get address from cert file or pem string",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var addr string
			var err error
			var isFile = utils.Exists(args[0])
			switch addressType {
			case addressTypeZXL:
				if isFile {
					addr, err = sdk.GetZXAddressFromCertPath(args[0])
					if err != nil {
						return err
					}
				} else {
					addr, err = sdk.GetZXAddressFromCertPEM(args[0])
					if err != nil {
						return err
					}
				}
			case addressTypeCM:
				if isFile {
					addr, err = sdk.GetCMAddressFromCertPath(args[0])
					if err != nil {
						return err
					}
				} else {
					addr, err = sdk.GetCMAddressFromCertPEM(args[0])
					if err != nil {
						return err
					}
				}
			case addressTypeEVM:
				if isFile {
					addr, err = sdk.GetEVMAddressFromCertPath(args[0])
					if err != nil {
						return err
					}
				} else {
					addr, err = sdk.GetEVMAddressFromCertBytes([]byte(args[0]))
					if err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unsupported address type %s", addressType)
			}

			output, err := prettyjson.Marshal(addr)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	util.AttachFlags(cmd, flags, []string{flagAddressType})
	return cmd
}

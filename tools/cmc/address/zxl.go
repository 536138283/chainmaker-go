// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package address

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	flagAddressType = "address-type"
	flagHashType    = "hash-type"

	// address types
	addressTypeZXL = "zxl"
	addressTypeCM  = "cm"
)

var (
	addressType string
	hashType    int
	hashAlgoMap = map[int]string{
		0: "SHA256",
		1: "SHA3_256",
		2: "SM3",
	}
)

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&addressType, flagAddressType, "cm", `The type of address obtained.
supported address types zhixinlian: zxl, chainmaker: cm 
eg. --address-type=zxl`)

	flags.IntVar(&hashType, flagHashType, 0,
		`The type of hash algo obtained. 0: SAH256 (default), 1: SHA3_256, 2: SM3"
eg. --address-type=0`)
}

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

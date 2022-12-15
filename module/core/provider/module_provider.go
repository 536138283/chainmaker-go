/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v3"
)

// CoreProvider core provider interface
type CoreProvider interface {
	// NewCoreEngine return core engine, error
	NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error)
}

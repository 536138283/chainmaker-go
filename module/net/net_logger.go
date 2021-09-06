package net

import (
	"chainmaker.org/chainmaker-go/logger"
	liquid "chainmaker.org/chainmaker/chainmaker-net-liquid/liquidnet"
	"chainmaker.org/chainmaker/protocol/v2"
)

var GlobalNetLogger protocol.Logger

func init() {
	GlobalNetLogger = logger.GetLogger(logger.MODULE_NET)
	liquid.InitLogger(GlobalNetLogger, func(chainId string) protocol.Logger {
		return logger.GetLoggerByChain(logger.MODULE_NET, chainId)
	})
}

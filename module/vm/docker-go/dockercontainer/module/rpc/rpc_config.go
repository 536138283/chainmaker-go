package rpc

import "time"

const (
	ServerMinInterval = time.Duration(1) * time.Minute
	ConnectionTimeout = 5 * time.Second

	DialTimeout        = 10 * time.Second
	MaxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	MaxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
)

package rpc

import "time"

const (
	ServerMinInterval = time.Duration(1) * time.Minute
	ConnectionTimeout = 5 * time.Second
)

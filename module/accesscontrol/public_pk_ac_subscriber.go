/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/hex"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"github.com/gogo/protobuf/proto"
)

var _ msgbus.Subscriber = (*pkACProvider)(nil)

// OnMessage contract event data is a []string, hexToString(proto.Marshal(data))
//  @Description:
//  @receiver p
//  @param msg
//
func (p *pkACProvider) OnMessage(msg *msgbus.Message) {
	p.log.Infof("[AC_PK] receive msg, topic: %s", msg.Topic.String())
	switch msg.Topic {
	case msgbus.ChainConfig:
		p.onMessageChainConfig(msg)
	case msgbus.MaxbftEpochConf:
		p.onMessageMaxbftChainconfigInEpoch(msg)
	}
}

// OnQuit
//  @Description: shut down
//  @receiver p
//,
func (p *pkACProvider) OnQuit() {

}

// onMessageChainConfig used to handle chain conf message
//  @Description:
//  @receiver p
//  @param msg
//
func (p *pkACProvider) onMessageChainConfig(msg *msgbus.Message) {
	dataStr, _ := msg.Payload.([]string)
	dataBytes, err := hex.DecodeString(dataStr[0])
	if err != nil {
		p.log.Error(err)
		return
	}
	chainConfig := &config.ChainConfig{}
	_ = proto.Unmarshal(dataBytes, chainConfig)

	// update chainconfig instantly
	p.messageChainConfig(chainConfig, false)
}

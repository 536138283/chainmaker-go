/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/os/gtimer"

	"chainmaker.org/chainmaker-go/logger"
	abftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/abft"
)

const (
	initBackOff = time.Millisecond * 200
	maxBackOff  = time.Second * 10

	defaultMsgChSize = 1000
)

type msgSender struct {
	logger *logger.CMLogger
	id     string
	mu     sync.Mutex
	seq    uint64
	events map[uint64]map[uint64]*gtimer.Entry
	timer  *gtimer.Timer
	msgCh  chan *abftpb.ABFTMessageReq
}

func newMsgSender(logger *logger.CMLogger, id string) *msgSender {
	return &msgSender{
		logger: logger,
		id:     id,
		seq:    1,
		events: make(map[uint64]map[uint64]*gtimer.Entry),
		timer:  gtimer.New(),
		msgCh:  make(chan *abftpb.ABFTMessageReq, defaultMsgChSize),
	}
}

func (m *msgSender) addMsg(msg *abftpb.ABFTMessageReq) {
	seq := atomic.AddUint64(&m.seq, 1)
	msg.Seq = seq

	m.retry(msg, seq, 0)
}

func (m *msgSender) retry(msg *abftpb.ABFTMessageReq, seq uint64, times int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	times += 1
	interval := backOffDelay(times)
	entry := m.timer.AddOnce(interval, gtimer.JobFunc(func() {
		m.retry(msg, seq, times)
	}))
	m.logger.Debugf("[%s] retry msg seq: %v, height: %v, to: %v, times: %v", m.id, seq, msg.Height, msg.To, times)

	if _, ok := m.events[msg.Height]; !ok {
		m.events[msg.Height] = make(map[uint64]*gtimer.Entry)
	}

	if e, ok := m.events[msg.Height][seq]; ok {
		e.Close()
	}
	m.events[msg.Height][seq] = entry

	m.msgCh <- msg
}

func (m *msgSender) ack(msg *abftpb.ABFTMessageRsp) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debugf("[%s] receive ack msg seq: %v, height: %v, from: %v, code: %s", m.id, msg.Seq, msg.Height, msg.From, msg.Code)
	seqEntry, ok := m.events[msg.Height]
	if !ok {
		m.logger.Warnf("[%s] receive ack can not find events seq: %v, height: %v with height", m.id, msg.Seq, msg.Height)
		return
	}

	entry, ok := seqEntry[msg.Seq]
	if !ok {
		m.logger.Warnf("[%s] receive ack can not find events seq: %v, height: %v with seq", m.id, msg.Seq, msg.Height)
		return
	}

	entry.Close()
	delete(seqEntry, msg.Seq)
}

func (m *msgSender) cleanHeight(height uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	seqEntry, ok := m.events[height]
	if !ok {
		m.logger.Warnf("can not clean events height: %v", height)
		return
	}

	for seq, entry := range seqEntry {
		entry.Close()
		delete(seqEntry, seq)
	}
}

func backOffDelay(n int) time.Duration {
	delay := initBackOff << n

	if delay > maxBackOff || delay < 0 {
		delay = maxBackOff
	}

	return delay
}

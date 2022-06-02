/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"chainmaker.org/chainmaker-go/net/p2p/libp2pgmtls"
	"chainmaker.org/chainmaker-go/net/p2p/libp2ptls"
	"chainmaker.org/chainmaker-go/net/p2p/revoke"
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
)

var readyC chan struct{}

type connNotifyUnit struct {
	c      network.Conn
	action bool // true: connected, false: disconnected
}

// networkNotify is an implementation of network.Notifiee.
var networkNotify = func(host *LibP2pHost) network.Notifiee {
	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			select {
			case <-host.ctx.Done():
				return
			case <-readyC:

			}
			host.connHandleOnce.Do(func() {
				go host.connHandleLoop()
			})
			host.connHandleC <- &connNotifyUnit{
				c:      c,
				action: true,
			}
		},
		DisconnectedF: func(_ network.Network, c network.Conn) {
			select {
			case <-host.ctx.Done():
				return
			case <-readyC:

			}
			host.connHandleOnce.Do(func() {
				go host.connHandleLoop()
			})
			host.connHandleC <- &connNotifyUnit{
				c:      c,
				action: false,
			}
		},
	}
}

// LibP2pHost is a libP2pHost which use libp2p as local net provider.
type LibP2pHost struct {
	startUp                      bool
	lock                         sync.Mutex
	ctx                          context.Context
	host                         host.Host                // host
	connManager                  *PeerConnManager         // connManager
	blackList                    *BlackList               // blackList
	revokedValidator             *revoke.RevokedValidator // revokedValidator
	peerStreamManager            *PeerStreamManager
	connSupervisor               *ConnSupervisor
	isTls                        bool
	isGmTls                      bool
	peerChainIdsRecorder         *PeerIdChainIdsRecorder
	newTlsPeerChainIdsNotifyC    chan map[string][]string
	removeTlsPeerNotifyC         chan string
	certPeerIdMapper             *CertIdPeerIdMapper
	newTlsCertIdPeerIdNotifyC    chan string
	removeTlsCertIdPeerIdNotifyC chan string
	peerIdTlsCertStore           *PeerIdTlsCertStore
	addPeerIdTlsCertNotifyC      chan map[string][]byte
	removePeerIdTlsCertNotifyC   chan string
	tlsChainTrustRoots           *libp2ptls.ChainTrustRoots
	gmTlsChainTrustRoots         *libp2pgmtls.ChainTrustRoots
	opts                         []libp2p.Option

	connHandleOnce sync.Once
	connHandleC    chan *connNotifyUnit
}

func (lh *LibP2pHost) initTlsCsAndSubassemblies() {
	lh.newTlsPeerChainIdsNotifyC = make(chan map[string][]string, 50)
	lh.removeTlsPeerNotifyC = make(chan string, 50)
	lh.newTlsCertIdPeerIdNotifyC = make(chan string, 50)
	lh.removeTlsCertIdPeerIdNotifyC = make(chan string, 50)
	lh.addPeerIdTlsCertNotifyC = make(chan map[string][]byte, 50)
	lh.removePeerIdTlsCertNotifyC = make(chan string, 50)
	lh.peerChainIdsRecorder = newPeerIdChainIdsRecorder(lh.newTlsPeerChainIdsNotifyC, lh.removeTlsPeerNotifyC)
	lh.certPeerIdMapper = newCertIdPeerIdMapper(lh.newTlsCertIdPeerIdNotifyC, lh.removeTlsCertIdPeerIdNotifyC)
	lh.peerIdTlsCertStore = newPeerIdTlsCertStore(lh.addPeerIdTlsCertNotifyC, lh.removePeerIdTlsCertNotifyC)
}

func (lh *LibP2pHost) connHandleLoop() {
	for {
		select {
		case <-lh.ctx.Done():
			return
		case u := <-lh.connHandleC:
			if u.action {
				// connected notify
				lh.peerStreamManager.initPeerStream(u.c.RemotePeer())
				pid := u.c.RemotePeer()
				lh.connManager.AddConn(pid, u.c)
				logger.Infof("[Host] new connection connected(remote peer-id:%s, remote multi-addr:%s)",
					u.c.RemotePeer().Pretty(), u.c.RemoteMultiaddr().String())
				continue
			}
			// disconnected notify
			// 判断连接是否有多个（单机libp2p可能建立多个地址的连接，例如127.0.0.1 192.168.XXX.XXX）
			// 如果连接有一个以上，不能删除节点的上层状态

			conn := lh.connManager.GetConns(u.c.RemotePeer())
			if len(conn) > 1 {
				// 不止一个连接
				lh.connManager.RemoveConn(u.c.RemotePeer(), u.c)
				logger.Infof("[Host] more than one connection, connection disconnected(remote peer-id:%s, remote multi-addr:%s)",
					u.c.RemotePeer().Pretty(), u.c.RemoteMultiaddr().String())
			} else {
				if conn != nil && conn[0].RemoteMultiaddr().String() != u.c.RemoteMultiaddr().String() {
					logger.Infof("[Host] connection disconnected failed, (remote peer-id:%s, remote multi-addr:%s, connection multi-addr:%s)",
						u.c.RemotePeer().Pretty(), u.c.RemoteMultiaddr().String(), conn[0].RemoteMultiaddr().String())
					return
				}
				logger.Infof("[Host] connection disconnected(remote peer-id:%s, remote multi-addr:%s)",
					u.c.RemotePeer().Pretty(), u.c.RemoteMultiaddr().String())
				pid := u.c.RemotePeer().Pretty()
				lh.connManager.RemoveConn(u.c.RemotePeer(), u.c)
				logger.Infof("[Host] remove connection done (remote peer-id:%s)", pid)
				if lh.removeTlsPeerNotifyC != nil {
					lh.removeTlsPeerNotifyC <- pid
					logger.Infof("[Host] remove peer from peer chain id map done (remote peer-id:%s)", pid)
				}
				if lh.removeTlsCertIdPeerIdNotifyC != nil {
					lh.removeTlsCertIdPeerIdNotifyC <- pid
					logger.Infof("[Host] remove peer from peer cert id map done (remote peer-id:%s)", pid)
				}
				if lh.removePeerIdTlsCertNotifyC != nil {
					lh.removePeerIdTlsCertNotifyC <- pid
					logger.Infof("[Host] remove peer from peer tls cert map done (remote peer-id:%s)", pid)
				}
				lh.peerStreamManager.cleanPeerStream(u.c.RemotePeer())
				logger.Infof("[Host] remove peer from peer stream manager map done (remote peer-id:%s)", pid)
			}
		}
	}
}

// PeerStreamManager
func (lh *LibP2pHost) PeerStreamManager() *PeerStreamManager {
	return lh.peerStreamManager
}

// Context
func (lh *LibP2pHost) Context() context.Context {
	return lh.ctx
}

// Host is libp2p.Host.
func (lh *LibP2pHost) Host() host.Host {
	return lh.host
}

// HasConnected return true if the peer which id is the peerId given has connected. Otherwise return false.
func (lh *LibP2pHost) HasConnected(peerId peer.ID) bool {
	return lh.connManager.IsConnected(peerId)
}

// IsRunning return true when libp2p has started up.Otherwise return false.
func (lh *LibP2pHost) IsRunning() bool {
	return lh.startUp
}

// NewLibP2pHost create new LibP2pHost instance.
func NewLibP2pHost(ctx context.Context) *LibP2pHost {
	return &LibP2pHost{
		startUp:          false,
		ctx:              ctx,
		connManager:      NewPeerConnManager(),
		blackList:        NewBlackList(),
		revokedValidator: revoke.NewRevokedValidator(),
		opts:             make([]libp2p.Option, 0),
		connHandleOnce:   sync.Once{},
		connHandleC:      make(chan *connNotifyUnit, 10),
	}
}

// Start libP2pHost.
func (lh *LibP2pHost) Start() error {
	lh.lock.Lock()
	defer lh.lock.Unlock()
	if lh.startUp {
		logger.Warn("[Host] host is running. ignored.")
		return nil
	}
	logger.Info("[Host] stating host...")
	node, err := libp2p.New(lh.ctx, lh.opts...)
	if err != nil {
		return err
	}
	lh.host = node
	// network notify
	node.Network().Notify(networkNotify(lh))
	logger.Info("[Host] host stated.")
	for _, addr := range node.Addrs() {
		logger.Infof("[Host] host listening on address:%s/p2p/%s", addr.String(), node.ID().Pretty())
	}
	if err := lh.handleTlsPeerChainIdsNotifyC(); err != nil {
		return err
	}
	if err := lh.handleTlsCertIdPeerIdNotifyC(); err != nil {
		return err
	}
	if err := lh.handlePeerIdTlsCertStoreNotifyC(); err != nil {
		return err
	}
	lh.startUp = true
	return nil
}

func (lh *LibP2pHost) handleTlsPeerChainIdsNotifyC() error {
	if lh.peerChainIdsRecorder != nil {
		if err := lh.peerChainIdsRecorder.handleNewTlsPeerChainIdsNotifyC(); err != nil {
			return err
		}
		if err := lh.peerChainIdsRecorder.handleRemoveTlsPeerNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

func (lh *LibP2pHost) handleTlsCertIdPeerIdNotifyC() error {
	if lh.certPeerIdMapper != nil {
		if err := lh.certPeerIdMapper.handleNewTlsCertIdPeerIdNotifyC(); err != nil {
			return err
		}
		if err := lh.certPeerIdMapper.handleRemoveTlsPeerNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

func (lh *LibP2pHost) handlePeerIdTlsCertStoreNotifyC() error {
	if lh.peerIdTlsCertStore != nil {
		if err := lh.peerIdTlsCertStore.startHandlingNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

// Stop libP2pHost.
func (lh *LibP2pHost) Stop() error {
	if lh.peerChainIdsRecorder != nil {
		lh.peerChainIdsRecorder.stopHandling()
	}
	if lh.certPeerIdMapper != nil {
		lh.certPeerIdMapper.stopHandling()
	}
	if lh.connSupervisor != nil {
		lh.connSupervisor.stopSupervising()
	}
	if lh.peerIdTlsCertStore != nil {
		lh.peerIdTlsCertStore.stopHandling()
	}
	lh.peerStreamManager.reset()
	return lh.host.Close()
}

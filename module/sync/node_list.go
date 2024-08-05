package sync

import (
	"sync"
	"time"

	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
)

type NodeState = syncPb.NodeState

type NodeList struct {
	mutex sync.Mutex
	nodes map[string]*NodeState
}

func NewNodeList() *NodeList {
	return &NodeList{
		nodes: make(map[string]*NodeState),
	}
}

func (nl *NodeList) AddNode(id string, height, archived uint64) {
	nl.mutex.Lock()
	defer nl.mutex.Unlock()
	nl.nodes[id] = &NodeState{
		NodeId:         id,
		Height:         height,
		ArchivedHeight: archived,
		ReceiveTime:    time.Now().Unix(),
	}
}

func (nl *NodeList) GetAll() []*NodeState {
	nl.mutex.Lock()
	defer nl.mutex.Unlock()
	nodes := make([]*NodeState, 0, len(nl.nodes))
	for _, node := range nl.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

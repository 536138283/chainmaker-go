package dockervm_interface

import "chainmaker.org/chainmaker/protocol"

type VmManager interface {
	// VmManager vm manager
	protocol.VmManager
	// StartDockerVM start docker vm
	StartDockerVM() error
	// StopDockerVM stop docker vm
	StopDockerVM() error
}

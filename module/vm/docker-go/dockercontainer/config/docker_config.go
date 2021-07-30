package config

const (

	// CGroupRoot cgroup location is not allow user to change
	CGroupRoot      = "/sys/fs/cgroup/memory/chainmaker"
	ProcsFile       = "cgroup.procs"
	MemoryLimitFile = "memory.limit_in_bytes"
	SwapLimitFile   = "memory.swappiness"
	RssLimit        = 500 // 10 MB

	DMSDir      = "/dms"
	DMSSockPath = "dms.sock"

	ContractsDir = "contracts"
	ShareDir     = "share"
	SockDir      = "sock"
	SockName     = "cdm.sock"
)

var (
	ContractBaseDir string
	ShareBaseDir    string
	SockBaseDir     string
	SandBoxTimeout  = 2

	EnablePProf = false
)

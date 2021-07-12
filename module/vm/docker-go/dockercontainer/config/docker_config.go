package config

const (
	CGroupRoot      = "/sys/fs/cgroup/memory/chainmaker"
	ProcsFile       = "cgroup.procs"
	MemoryLimitFile = "memory.limit_in_bytes"
	SwapLimitFile   = "memory.swappiness"

	RssLimit = 500 // 10 MB
	UserNum  = 600

	TimeLimit = 300     // process running time limit
	Port      = "12355" // Port for chainmaker and docker manager -- cdm_rpc

	SockPath = "/uds.sock" // sock file for sandbox and docker manager -- dms_rpc

	MountDir = "/mount"

	// LogFile log info
	LogFile          = "/docker_manager.log"
	DisplayInConsole = true
	ShowLine         = false
	//LogLevel         = "DEBUG"
	LogLevel = "INFO" //
)

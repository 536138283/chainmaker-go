package config

const (
	CGroupRoot      = "/sys/fs/cgroup/memory/chainmaker"
	ProcsFile       = "cgroup.procs"
	MemoryLimitFile = "memory.limit_in_bytes"
	SwapLimitFile   = "memory.swappiness"

	RssLimit = 500 // 10 MB
	UserNum  = 5

	TimeLimit = 300 // process running time limit
	Port      = "12355"
)

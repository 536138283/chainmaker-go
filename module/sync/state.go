package sync

type state struct {
	//max height of block has been synced
	blocks_has_synced uint64
	//num of block in cache waiting to be committed
	blocks_in_cache int
}

type getStateFn func() state

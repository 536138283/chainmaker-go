package common

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_ReentrantLock(t *testing.T) {
	lock := &ReentrantLocks{
		ReentrantLocks: make(map[string]interface{}),
		Mu:             sync.Mutex{},
	}

	for i := 0; i < 3; i++ {
		go func(i int) {
			if lock.Lock("") {
				require.False(t, lock.Lock(""))
				defer lock.Unlock("")
				fmt.Printf("%d get lock \n", i)
				time.Sleep(2 * time.Second)
			}
		}(i)
	}

	for i := 0; i < 3; i++ {
		go func(i int) {
			for {
				if lock.Lock("") {
					defer lock.Unlock("")
					fmt.Printf("finally %d get lock \n", i)
					break
				}
			}
		}(i)
	}

	time.Sleep(5 * time.Second)
}

func Test_ReentrantLocks(t *testing.T) {
	locks := &ReentrantLocks{
		ReentrantLocks: make(map[string]interface{}),
		Mu:             sync.Mutex{},
	}
	for i := 0; i < 3; i++ {
		go func(i int) {
			if locks.Lock("1") {
				require.False(t, locks.Lock("1"))
				defer locks.Unlock("1")
				fmt.Printf("%d get lock", i)
				time.Sleep(2 * time.Second)
			}
		}(i)
	}

	for i := 0; i < 3; i++ {
		go func(i int) {
			for {
				if locks.Lock("2") {
					defer locks.Unlock("2")
					fmt.Printf("finally %d get lock \n", i)
					time.Sleep(1 * time.Second)
					break
				}
			}
		}(i)
	}
	time.Sleep(5 * time.Second)

}

type reentrantLock struct {
	reentrantLock *int32
}

func (l *reentrantLock) lock(key string) bool {
	return atomic.CompareAndSwapInt32(l.reentrantLock, 0, 1)
}

func (l *reentrantLock) unlock(key string) bool {
	return atomic.CompareAndSwapInt32(l.reentrantLock, 1, 0)
}

//func TestCuckooFilterLongRun(t *testing.T) {
//	cf := cuckoo.NewFilter(4, 32, 99999999, cuckoo.TableTypePacked)
//	var count uint64
//	var dupCount uint64
//	atomic.AddUint64(&count, 0)
//	atomic.AddUint64(&dupCount, 0)
//	var loopSize uint64 = 1000000
//
//	t0 := CurrentTimeMillisSeconds()
//	for {
//		txid := uuid.GetUUID()
//		txidBytes := []byte(txid)
//		if cf.Contain(txidBytes) {
//			atomic.AddUint64(&dupCount, 1)
//			break
//		}
//		if !cf.Add(txidBytes) {
//			break
//		}
//		c := atomic.AddUint64(&count, 1)
//
//		if c % loopSize == 0 {
//			t1 := CurrentTimeMillisSeconds()
//			cfBytes, _ := cf.Encode()
//			t2 := CurrentTimeMillisSeconds()
//			cf, _ = cuckoo.Decode(cfBytes)
//			t3 := CurrentTimeMillisSeconds()
//			fmt.Println(fmt.Sprintf("count(%d, %d) = (dup:%d/%d, size:%d, total:%d, en:%d, de:%d) %v",
//				c, cf.Size(), dupCount, loopSize, len(cfBytes), CurrentTimeMillisSeconds()-t0, t2-t1, t3-t2, cf.Contain(txidBytes)))
//			t0 = CurrentTimeMillisSeconds()
//			dupCount = 0
//		}
//	}
//	fmt.Println(cf.Size())
//}

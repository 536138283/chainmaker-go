package hbbft

import (
	"fmt"
	"testing"
)

func TestCommitter_Commit(t *testing.T) {

	commiter := initCommiter()

	fmt.Println(commiter)

}

func initCommiter() *Committer {

	coreExecute := NewCoreExecute(nil)
	packager := NewPackager(coreExecute)

	return NewCommitter(coreExecute,packager)
}
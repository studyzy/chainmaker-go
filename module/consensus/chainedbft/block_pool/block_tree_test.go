package blockpool

import (
	"testing"

	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

func TestBlockTree_InsertBlock(t *testing.T) {
	rootBlk := common.Block{}
	tree := NewBlockTree(&rootBlk, 10)
	//tree.InsertBlock()
	_ = tree
}

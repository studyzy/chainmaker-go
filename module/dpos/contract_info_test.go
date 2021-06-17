package dpos

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker-go/vm/native"
)

func TestGetStakeAddr(t *testing.T) {
	fmt.Println(native.StakeContractAddr())
}

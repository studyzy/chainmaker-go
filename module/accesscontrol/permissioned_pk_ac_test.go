package accesscontrol

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
)

func TestParsePublicKey(t *testing.T) {
	sk, err := asym.PrivateKeyFromPEM([]byte(TestSK1), nil)
	if err != nil {
		fmt.Println(err)
	}
	commonNodeId, err := helper.CreateLibp2pPeerIdWithPublicKey(sk.PublicKey())
	if err != nil {
		fmt.Println(err)
	}
	pk, err := asym.PublicKeyFromPEM([]byte(TestPK1))
	if err != nil {
		fmt.Println(err)
	}
	openNodeId, err := helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("common:", commonNodeId)
	fmt.Println("open:", openNodeId)
}

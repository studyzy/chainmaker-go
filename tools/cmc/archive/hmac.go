package archive

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
)

// sm3(Fchain_id+Fblock_height+sm3(Fblock_with_rwset)+key)
func Hmac() (string, error) {
	bz, err := hash.Get(crypto.HASH_TYPE_SM3, []byte("Fchain_id+Fblock_height+sm3(Fblock_with_rwset)+key"))
	if err != nil {
		return "", err
	}
	fmt.Printf("%x\n", bz)
	hexSum := hex.EncodeToString(bz)
	fmt.Println("hexSum=", hexSum)
	return hexSum, nil
}

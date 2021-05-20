package archive

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
)

func sm3(data []byte) (string, error) {
	bz, err := hash.Get(crypto.HASH_TYPE_SM3, data)
	if err != nil {
		return "", err
	}
	fmt.Printf("%x\n", bz)
	hexSum := hex.EncodeToString(bz)
	fmt.Println("hexSum=", hexSum)
	return hexSum, nil
}

// sm3(Fchain_id+Fblock_height+sm3(Fblock_with_rwset)+key)
func Hmac(chainId, blkHeight, sumBlkWithRWSet, secretKey []byte) (string, error) {
	var data []byte
	data = append(data, chainId...)
	data = append(data, blkHeight...)
	data = append(data, sumBlkWithRWSet...)
	data = append(data, secretKey...)
	return sm3(data)
}

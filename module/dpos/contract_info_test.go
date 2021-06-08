package dpos

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBigBytes(t *testing.T) {
	for i := 0; i < 100; i++ {
		rand.Seed(time.Now().Unix() + int64(i*10))
		num := big.NewInt(0)
		num, ok := num.SetString(fmt.Sprintf("%d", rand.Uint64()), 10)
		require.True(t, ok)
		require.EqualValues(t, num.Bytes(), []byte(num.String()))
	}
}

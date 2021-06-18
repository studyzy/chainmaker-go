package dpos

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesEqual(t *testing.T) {
	bz := make([]byte, 0)
	require.True(t, bytes.Equal(bz, nil))
}

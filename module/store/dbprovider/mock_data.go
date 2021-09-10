/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dbprovider

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
)

var keyLen int

const hitKeyFormat = "%016d+"
const missingKeyFormat = "%016d-"

func init() {
	var b bytes.Buffer
	keyLen, _ = fmt.Fprintf(&b, hitKeyFormat, math.MaxInt32)
	b.Reset()
	missingKeyLen, _ := fmt.Fprintf(&b, missingKeyFormat, math.MaxInt32)
	if keyLen != missingKeyLen {
		panic("len(key) != len(missingKey)")
	}
}

type keyGenerator interface {
	NKey() int
	Key(i int) []byte
}

type EntryGenerator interface {
	keyGenerator
	Value(i int) []byte
}

type pairedEntryGenerator struct {
	keyGenerator
	randomValueGenerator
}

type randomValueGenerator struct {
	b []byte
	k int
}

func (g *randomValueGenerator) Value(i int) []byte {
	i = (i * g.k) % len(g.b)
	return g.b[i : i+g.k]
}

type predefinedKeyGenerator struct {
	keys [][]byte
}

func (g *predefinedKeyGenerator) NKey() int {
	return len(g.keys)
}

func (g *predefinedKeyGenerator) Key(i int) []byte {
	return g.keys[i]
}

func newFullRandomKeys(n int, start int, format string) [][]byte {
	keys := newSequentialKeys(n, start, format)
	r := rand.New(rand.NewSource(time.Now().Unix())) //nolint: gosec
	for i := 0; i < n; i++ {
		j := r.Intn(n)
		keys[i], keys[j] = keys[j], keys[i]
	}
	return keys
}

func newSequentialKeys(n int, start int, keyFormat string) [][]byte {
	keys := make([][]byte, n)
	buffer := make([]byte, n*keyLen)
	for i := 0; i < n; i++ {
		begin, end := i*keyLen, (i+1)*keyLen
		key := buffer[begin:begin:end]
		n, _ := fmt.Fprintf(bytes.NewBuffer(key), keyFormat, start+i)
		if n != keyLen {
			panic("n != keyLen")
		}
		keys[i] = buffer[begin:end:end]
	}
	return keys
}

func newFullRandomKeyGenerator(start, n int) keyGenerator {
	return &predefinedKeyGenerator{keys: newFullRandomKeys(n, start, hitKeyFormat)}
}

func makeRandomValueGenerator(r *rand.Rand, ratio float64, valueSize int) randomValueGenerator {
	b := compressibleBytes(r, ratio, valueSize)
	max := maxInt(valueSize, 1024*1024)
	for len(b) < max {
		b = append(b, compressibleBytes(r, ratio, valueSize)...)
	}
	return randomValueGenerator{b: b, k: valueSize}
}

func compressibleBytes(r *rand.Rand, ratio float64, n int) []byte { //nolint: gosec
	m := maxInt(int(float64(n)*ratio), 1)
	p := randomBytes(r, m)
	b := make([]byte, 0, n+n%m)
	for len(b) < n {
		b = append(b, p...)
	}
	return b[:n]
}

func maxInt(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func randomBytes(r *rand.Rand, n int) []byte { //nolint: gosec
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = ' ' + byte(r.Intn('~'-' '+1))
	}
	return b
}

func NewFullRandomEntryGenerator(start, n int) EntryGenerator {
	r := rand.New(rand.NewSource(time.Now().Unix())) //nolint: gosec
	return &pairedEntryGenerator{
		keyGenerator:         newFullRandomKeyGenerator(start, n),
		randomValueGenerator: makeRandomValueGenerator(r, 0.5, 100),
	}
}

func GetMockDBConfig(path string) *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	if path == "" {
		path = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	}
	conf.StorePath = path

	lvlConfig := make(map[string]interface{})
	lvlConfig["store_path"] = path
	//rocksConfig := &localconf.RocksDbConfig{
	//	StorePath: path,
	//}
	dbConfig := &localconf.DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lvlConfig,
		//RocksDbConfig: rocksConfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig
	conf.DisableContractEventDB = true
	return conf
}

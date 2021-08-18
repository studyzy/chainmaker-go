/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"chainmaker.org/chainmaker-go/logger"
	"go.etcd.io/etcd/raft/v3"

	"go.etcd.io/etcd/client/pkg/v3/fileutil"
)

func Test_computeUpdatedNodes(t *testing.T) {
	type args struct {
		oldSet []uint64
		newSet []uint64
	}
	tests := []struct {
		name        string
		args        args
		wantRemoved []uint64
		wantAdded   []uint64
	}{
		{
			"no change",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 2, 3},
			},
			[]uint64{},
			[]uint64{},
		},
		{
			"add 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 2, 3, 4, 5},
			},
			[]uint64{},
			[]uint64{4, 5},
		},
		{
			"remove 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1},
			},
			[]uint64{2, 3},
			[]uint64{},
		},
		{
			"add 2 nodes and remove 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 4, 5},
			},
			[]uint64{2, 3},
			[]uint64{4, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemoved, gotAdded := computeUpdatedNodes(tt.args.oldSet, tt.args.newSet)
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("computeUpdatedNodes() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("computeUpdatedNodes() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
		})
	}
}

/**
 * 测试Purge File前，需要在PurgeFile方法return前添加代码 	time.Sleep(time.Millisecond * 200)
 * 否则，测试完成后，协程未来得及删除测试文件。
 */
func TestPurgeFile(t *testing.T) {
	storagePath := t.TempDir()
	t.Logf("File Dir : %s", storagePath)
	//创建5个WAl文件
	for i := 10; i < 15; i++ {
		index := i*2 + 5
		p := filepath.Join(storagePath, fmt.Sprintf("%016d-%016d.wal", i, index))
		f, err := fileutil.LockFile(p, os.O_WRONLY|os.O_CREATE, fileutil.PrivateFileMode)
		if err != nil {
			t.Errorf("Lock File Failed : %s", err)
		}

		if err = fileutil.Preallocate(f.File, 64*1000, true); err != nil {
			t.Errorf("Preallocate an initial WAL file Failed : %xs", err)
		}
		_ = f.Close()
	}

	config := ConsensusRaftImplConfig{
		ChainID: "chain1",
		NodeId:  "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35",
	}
	consensus, err := New(config)
	if err != nil {
		t.Errorf("New Raft Failed : %s", err)
	}

	consensus.PurgeFile(storagePath)

	fileNames, err := fileutil.ReadDir(storagePath)
	if err != nil {
		t.Errorf("Read Path Failed : %s", err)
	}
	want := 1
	fileCount := len(fileNames)
	if !reflect.DeepEqual(fileCount, want) {
		t.Errorf("WAL File Purged, Path File Count = %v, want %v", fileCount, want)
	}

}

var log = logger.GetLoggerByChain(logger.MODULE_CONSENSUS, "chainID")
var id = "4b77737245594276"
var reg = "[%x] receive from raft ready, %v"
var ready = raft.Ready{}
var readyStr = `eyJ0ZXJtIjowLCJ2b3RlIjowLCJjb21taXQiOjAsIlJlYWRTdGF0ZXMiOm51bGwsIkVudHJpZXMiOm51bGwsIlNuYXBzaG90Ijp7Im1
ldGFkYXRhIjp7ImNvbmZfc3RhdGUiOnsiYXV0b19sZWF2ZSI6ZmFsc2V9LCJpbmRleCI6MCwidGVybSI6MH19LCJDb21taXR0ZWRFbnRyaWVzIjpudWxsLC
JNZXNzYWdlcyI6W3sidHlwZSI6MywidG8iOjM2OTY3MzU0MDMyMTYwOTg2MzUsImZyb20iOjU2NTA0MTE2NDU3NDM1NTY0NzAsInRlcm0iOjIsImxvZ1Rlc
m0iOjIsImluZGV4Ijo1LCJlbnRyaWVzIjpudWxsLCJjb21taXQiOjQsInNuYXBzaG90Ijp7Im1ldGFkYXRhIjp7ImNvbmZfc3RhdGUiOnsiYXV0b19sZWF2
ZSI6ZmFsc2V9LCJpbmRleCI6MCwidGVybSI6MH19LCJyZWplY3QiOmZhbHNlLCJyZWplY3RIaW50IjowfV0sIk11c3RTeW5jIjpmYWxzZX0`

func BenchmarkDebugF(b *testing.B) {
	b.ResetTimer()
	decoded, _ := base64.StdEncoding.DecodeString(readyStr)
	_ = json.Unmarshal(decoded, &ready)
	for i := 0; i < b.N; i++ {
		log.Debugf(reg, id, describeReady(ready))
	}
}

func BenchmarkDebugDynamic(b *testing.B) {
	b.ResetTimer()
	decoded, _ := base64.StdEncoding.DecodeString(readyStr)
	_ = json.Unmarshal(decoded, &ready)
	for i := 0; i < b.N; i++ {
		log.DebugDynamic(func() string {
			return fmt.Sprintf(reg, id, describeReady(ready))
		})
	}
}

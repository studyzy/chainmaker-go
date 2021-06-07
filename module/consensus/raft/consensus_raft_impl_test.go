/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"reflect"
	"testing"

	etcdraft "go.etcd.io/etcd/raft/v3"
)

func Test_computeUpdatedNodes(t *testing.T) {
	type args struct {
		oldSet []etcdraft.Peer
		newSet []etcdraft.Peer
	}
	tests := []struct {
		name        string
		args        args
		wantRemoved []etcdraft.Peer
		wantAdded   []etcdraft.Peer
	}{
		{
			"no change",
			args{
				oldSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}},
				newSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}},
			},
			[]etcdraft.Peer{},
			[]etcdraft.Peer{},
		},
		{
			"add 2 nodes",
			args{
				oldSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}},
				newSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}},
			},
			[]etcdraft.Peer{},
			[]etcdraft.Peer{{ID: 4}, {ID: 5}},
		},
		{
			"remove 2 nodes",
			args{
				oldSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}},
				newSet: []etcdraft.Peer{{ID: 1}},
			},
			[]etcdraft.Peer{{ID: 2}, {ID: 3}},
			[]etcdraft.Peer{},
		},
		{
			"add 2 nodes and remove 2 nodes",
			args{
				oldSet: []etcdraft.Peer{{ID: 1}, {ID: 2}, {ID: 3}},
				newSet: []etcdraft.Peer{{ID: 1}, {ID: 4}, {ID: 5}},
			},
			[]etcdraft.Peer{{ID: 2}, {ID: 3}},
			[]etcdraft.Peer{{ID: 4}, {ID: 5}},
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

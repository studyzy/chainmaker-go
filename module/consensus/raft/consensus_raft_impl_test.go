/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"reflect"
	"testing"
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

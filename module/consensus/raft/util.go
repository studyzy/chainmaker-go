/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"bytes"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// This file is copied from etcd/raft/util.go, and modified for more beautiful format

func entryFormatter(data []byte) string {
	if len(data) == 0 {
		return "empty entry"
	}
	block := new(common.Block)
	mustUnmarshal(data, block)
	return fmt.Sprintf("block(%d-%x)", block.Header.BlockHeight, block.Header.BlockHash)
}

func describeReady(rd raft.Ready) string {
	var buf strings.Builder
	if rd.SoftState != nil {
		fmt.Fprintf(&buf, "SoftState: %v, ", describeSoftState(*rd.SoftState))
	}
	if !raft.IsEmptyHardState(rd.HardState) {
		fmt.Fprintf(&buf, "HardState: %v, ", describeHardState(rd.HardState))
	}
	if len(rd.ReadStates) > 0 {
		fmt.Fprintf(&buf, "ReadStates: {%v}, ", rd.ReadStates)
	}
	if len(rd.Entries) > 0 {
		buf.WriteString("Entries: [")
		for i, e := range rd.Entries {
			fmt.Fprintf(&buf, "%v", describeEntry(e))
			if i < len(rd.Entries)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("], ")
	}
	if !raft.IsEmptySnap(rd.Snapshot) {
		fmt.Fprintf(&buf, "Snapshot: %s, ", describeSnapshot(rd.Snapshot))
	}
	if len(rd.CommittedEntries) > 0 {
		buf.WriteString("CommittedEntries: [")
		for i, e := range rd.CommittedEntries {
			fmt.Fprintf(&buf, "%v", describeEntry(e))
			if i < len(rd.Entries)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("], ")
	}
	if len(rd.Messages) > 0 {
		buf.WriteString("Messages: [")
		for i, msg := range rd.Messages {
			fmt.Fprintf(&buf, "%v", describeMessage(msg))
			if i < len(rd.Messages)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteString("]")
	}
	if buf.Len() > 0 {
		return strings.TrimSuffix(buf.String(), ", ")
	}
	return "<empty Ready>"
}

func describeSoftState(ss raft.SoftState) string {
	return fmt.Sprintf("{Lead:%x State:%s}", ss.Lead, ss.RaftState)
}

func describeHardState(hs raftpb.HardState) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "{Term:%d", hs.Term)
	if hs.Vote != 0 {
		fmt.Fprintf(&buf, " Vote:%x", hs.Vote)
	}
	fmt.Fprintf(&buf, " Commit:%d}", hs.Commit)
	return buf.String()
}

func describeEntry(e raftpb.Entry) string {
	formatConfChange := func(cc raftpb.ConfChangeI) string {
		return confChangesToString(cc.AsV2().Changes)
	}

	var formatted string
	switch e.Type {
	case raftpb.EntryNormal:
		formatted = entryFormatter(e.Data)
	case raftpb.EntryConfChange:
		var cc raftpb.ConfChange
		if err := cc.Unmarshal(e.Data); err != nil {
			formatted = err.Error()
		} else {
			formatted = formatConfChange(cc)
		}
	case raftpb.EntryConfChangeV2:
		var cc raftpb.ConfChangeV2
		if err := cc.Unmarshal(e.Data); err != nil {
			formatted = err.Error()
		} else {
			formatted = formatConfChange(cc)
		}
	}
	if formatted != "" {
		formatted = " " + formatted
	}
	return fmt.Sprintf("{%d/%d %s%s}", e.Term, e.Index, e.Type, formatted)
}

func confChangesToString(ccs []raftpb.ConfChangeSingle) string {
	var buf strings.Builder
	for i, cc := range ccs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		switch cc.Type {
		case raftpb.ConfChangeAddNode:
			buf.WriteString("AddNode ")
		case raftpb.ConfChangeAddLearnerNode:
			buf.WriteString("AddLearnerNode ")
		case raftpb.ConfChangeRemoveNode:
			buf.WriteString("RemoveNode ")
		case raftpb.ConfChangeUpdateNode:
			buf.WriteString("UpdateNode ")
		default:
			buf.WriteString("unknown")
		}
		fmt.Fprintf(&buf, "%x", cc.NodeID)
	}
	return buf.String()
}

func describeMessage(m raftpb.Message) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{%x->%x %v Term:%d Log:%d/%d", m.From, m.To, m.Type, m.Term, m.LogTerm, m.Index)
	if m.Reject {
		fmt.Fprintf(&buf, " Rejected (Hint: %d)", m.RejectHint)
	}
	if m.Commit != 0 {
		fmt.Fprintf(&buf, " Commit:%d", m.Commit)
	}
	if len(m.Entries) > 0 {
		fmt.Fprintf(&buf, " Entries:[")
		for i, e := range m.Entries {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(describeEntry(e))
		}
		fmt.Fprintf(&buf, "]")
	}
	if !raft.IsEmptySnap(m.Snapshot) {
		fmt.Fprintf(&buf, " Snapshot: %s", describeSnapshot(m.Snapshot))
	}
	fmt.Fprint(&buf, "}")
	return buf.String()
}

func describeNodes(nodes []uint64) string {
	var buf strings.Builder

	fmt.Fprint(&buf, "[")
	for i, node := range nodes {
		fmt.Fprintf(&buf, "%x", node)
		if i < len(nodes) {
			fmt.Fprint(&buf, ", ")
		}
	}
	fmt.Fprint(&buf, "]")

	return buf.String()
}

func describeConfState(state raftpb.ConfState) string {
	return fmt.Sprintf(
		"{Voters:%v VotersOutgoing:%v Learners:%v LearnersNext:%v AutoLeave:%v}",
		describeNodes(state.Voters), describeNodes(state.VotersOutgoing),
		describeNodes(state.Learners), describeNodes(state.LearnersNext), state.AutoLeave)
}

func describeSnapshot(snap raftpb.Snapshot) string {
	m := snap.Metadata
	return fmt.Sprintf("{Index:%d Term:%d ConfState:%s}", m.Index, m.Term, describeConfState(m.ConfState))
}

func describeConfChange(cc raftpb.ConfChange) string {
	return confChangesToString(cc.AsV2().Changes)
}

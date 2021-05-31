/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package p2p implement a p2p net service provider.
package p2p

import (
	"bytes"
	rootLog "chainmaker.org/chainmaker-go/logger"
	netPb "chainmaker.org/chainmaker/pb-go/net"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"os"
	"sync"
)

const (
	// DefaultLibp2pListenAddress is the default address that libp2p will listen on.
	DefaultLibp2pListenAddress = "/ip4/0.0.0.0/tcp/0"
	// DefaultLibp2pServiceTag is the default service tag for discovery service finding.
	DefaultLibp2pServiceTag = "chainmaker-libp2p-net"
	// DefaultLibp2pPubSubMaxMessageSize is the default max message size for pub-sub.
	DefaultLibp2pPubSubMaxMessageSize = 50 << 20
)

var logger = rootLog.GetLogger(rootLog.MODULE_NET)

// StringMapList is a string list using a perfect map
type StringMapList struct {
	lock    sync.Mutex
	mapList map[string]struct{}
}

// NewStringMapList creates a new StringMapList
func NewStringMapList() *StringMapList {
	return &StringMapList{
		mapList: make(map[string]struct{}),
	}
}

func (b *StringMapList) Remove(p string) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.mapList[p]; ok {
		delete(b.mapList, p)
	}
	return true
}

func (b *StringMapList) Add(p string) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.mapList[p] = struct{}{}
	return true
}

func (b *StringMapList) Contains(p string) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	_, ok := b.mapList[p]
	return ok
}

func (b *StringMapList) Size() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return len(b.mapList)
}

func (b *StringMapList) List() []string {
	b.lock.Lock()
	defer b.lock.Unlock()
	l := make([]string, 0)
	for p := range b.mapList {
		l = append(l, p)
	}
	return l
}

// fileExist 判断文件是否存在
func fileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}

// ParseMultiAddrs parse multi addr string to multiaddr.Multiaddr .
func ParseMultiAddrs(addrs []string) ([]multiaddr.Multiaddr, error) {
	var mutiAddrs = make([]multiaddr.Multiaddr, 0, len(addrs))
	if len(addrs) > 0 {
		for _, addr := range addrs {
			ma, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, err
			}
			mutiAddrs = append(mutiAddrs, ma)
		}
	}
	return mutiAddrs, nil
}

// ParseAddrInfo parse multi addr string to peer.AddrInfo .
func ParseAddrInfo(addrs []string) ([]peer.AddrInfo, error) {
	ais := make([]peer.AddrInfo, 0)
	mas, err := ParseMultiAddrs(addrs)
	if err != nil {
		return nil, err
	}
	for _, peerAddr := range mas {
		pif, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, err
		}
		ais = append(ais, *pif)
	}
	return ais, nil
}

func NewMsg(netMsg *netPb.NetMsg, chainId string, msgFlag string) *netPb.Msg {
	return &netPb.Msg{Msg: netMsg, ChainId: chainId, Flag: msgFlag}
}

//整形转换成8字节
func IntToBytes(n int) []byte {
	x := int64(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//8字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int64
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}

// FibonacciArray create a fibonacci array with length n.
func FibonacciArray(n int) []int64 {
	res := make([]int64, n, n)
	for i := 0; i < n; i++ {
		if i <= 1 {
			res[i] = 1
		} else {
			res[i] = res[i-1] + res[i-2]
		}
	}
	return res
}

/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

// nolint: revive
func TestNetService(t *testing.T) {
	certPath := filepath.Join("./testdata/cert")
	pidPath := filepath.Join("./testdata/pid")
	defer func() {
		_ = filepath.Walk(filepath.Join("./"), func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && strings.Contains(path, "default.log") {
				_ = os.Remove(path)
			}
			return nil
		})
	}()
	caBytes6666, err := ioutil.ReadFile(filepath.Join(certPath, "ca1.crt"))
	require.Nil(t, err)
	caBytes7777, err := ioutil.ReadFile(filepath.Join(certPath, "ca2.crt"))
	require.Nil(t, err)
	key1Path := filepath.Join(certPath, "key1.key")
	cert1Path := filepath.Join(certPath, "cert1.crt")
	pid1Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "pid1.nodeid"))
	require.Nil(t, err)
	pid1 := string(pid1Bytes)
	key2Path := filepath.Join(certPath, "key2.key")
	cert2Path := filepath.Join(certPath, "cert2.crt")
	pid2Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "pid2.nodeid"))
	require.Nil(t, err)
	pid2 := string(pid2Bytes)
	caBytes6666, err = ioutil.ReadFile(filepath.Join(certPath, "ca1.crt"))
	require.Nil(t, err)
	caBytes7777, err = ioutil.ReadFile(filepath.Join(certPath, "ca2.crt"))
	require.Nil(t, err)

	readyC := make(chan struct{})

	// start node A
	var nf NetFactory

	a, err := nf.NewNet(
		protocol.Libp2p,
		WithReadySignalC(readyC),
		WithListenAddr("/ip4/127.0.0.1/tcp/6666"),
		WithCrypto(false, key1Path, cert1Path),
	)
	require.Nil(t, err)
	//a.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + pid2)
	a.SetChainCustomTrustRoots(chainId1, [][]byte{caBytes6666, caBytes7777})

	err = a.Start()
	require.Nil(t, err)
	fmt.Println("node A is running...")

	var nsf NetServiceFactory
	nsa, err := nsf.NewNetService(
		a,
		chainId1,
		nil,
		nil,
		WithConsensusNodeUid(pid2),
	)
	require.Nil(t, err)
	err = nsa.Start()
	require.Nil(t, err)
	fmt.Println("net service A is running...")
	// start node B
	b, err := nf.NewNet(
		protocol.Libp2p,
		WithReadySignalC(readyC),
		WithListenAddr("/ip4/127.0.0.1/tcp/7777"),
		WithCrypto(false, key2Path, cert2Path),
	)
	require.Nil(t, err)
	err = b.AddSeed("/ip4/127.0.0.1/tcp/6666/p2p/" + pid1)
	require.Nil(t, err)
	b.SetChainCustomTrustRoots(chainId1, [][]byte{caBytes6666, caBytes7777})
	err = b.Start()
	require.Nil(t, err)
	fmt.Println("node B is running...")

	nsb, err := nsf.NewNetService(
		b,
		chainId1,
		nil,
		nil,
		WithConsensusNodeUid(pid1),
	)
	require.Nil(t, err)
	err = nsb.Start()
	require.Nil(t, err)
	fmt.Println("net service B is running...")

	close(readyC)

	// test A send msg to B
	data := []byte("hello")
	toNodeB := pid2
	passChan := make(chan bool)
	recHandlerB := func(id string, msg []byte, _ netPb.NetMsg_MsgType) error {
		fmt.Println("[B][chain1] recv a msg from peer[", id, "], msg：", string(msg))
		passChan <- true
		return nil
	}
	err = nsb.ReceiveMsg(netPb.NetMsg_TX, recHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B register receive msg handler for chain1")
	go func() {
		fmt.Println("[A]A send msg to B in chain1")
		for {
			if err = nsa.SendMsg(data, netPb.NetMsg_TX, toNodeB); err != nil {
				fmt.Println(err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}()
	timer := time.NewTimer(time.Minute)
	select {
	case <-timer.C:
		fmt.Println("==== test A send msg to B timeout ====")
		t.Fatal("test A send msg to B timeout")
	case <-passChan:
		fmt.Println("==== test A send msg to B pass ====")
	}

	subHandlerB := func(_ string, msg []byte, _ netPb.NetMsg_MsgType) error {
		fmt.Println("[B][chain1] recv a sub msg chain1：", string(msg))
		passChan <- true
		return nil
	}
	//// test broadcast
	//err = nsb.Subscribe(netPb.NetMsg_TX, subHandlerB)
	//require.Nil(t, err)
	//fmt.Println("[B]B subscribe topic of chain1")
	//
	//err = nsa.BroadcastMsg(data, netPb.NetMsg_TX)
	//require.Nil(t, err)
	//fmt.Println("[A]A broadcast a msg to chain1:", string(data))
	//
	//timer = time.NewTimer(time.Minute)
	//select {
	//case <-timer.C:
	//	fmt.Println("==== test broadcast timeout ====")
	//	t.Fatal("test broadcast failed")
	//case <-passChan:
	//	fmt.Println("==== test broadcast pass ====")
	//}
	//
	//// test cancel broadcast
	//err = nsb.CancelSubscribe(netPb.NetMsg_TX)
	//require.Nil(t, err)
	//fmt.Println("[B]B cancel subscribe topic of chain1")
	//
	//err = nsa.BroadcastMsg(data, netPb.NetMsg_TX)
	//require.Nil(t, err)
	//fmt.Println("[A]A broadcast a msg to chain1:", string(data))
	//
	//timer = time.NewTimer(10 * time.Second)
	//select {
	//case <-timer.C:
	//	fmt.Println("==== test cancel broadcast pass ====")
	//case <-passChan:
	//	fmt.Println("==== test cancel broadcast failed ====")
	//	t.Fatal("test cancel broadcast failed")
	//}

	// test consensus broadcast
	err = nsb.ConsensusSubscribe(netPb.NetMsg_TX, subHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B subscribe consensus topic of chain1")

	err = nsa.ConsensusBroadcastMsg(data, netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[A]A broadcast a msg to chain1 consensus nodes:", string(data))

	timer = time.NewTimer(time.Minute)
	select {
	case <-timer.C:
		fmt.Println("==== test consensus broadcast timeout ====")
		t.Fatal("test consensus broadcast failed")
	case <-passChan:
		fmt.Println("==== test consensus broadcast pass ====")
	}

	// test cancel consensus broadcast
	err = nsb.CancelConsensusSubscribe(netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[B]B cancel subscribe consensus topic of chain1")

	err = nsa.ConsensusBroadcastMsg(data, netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[A]A broadcast a msg to chain1 consensus nodes:", string(data))

	timer = time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("==== test cancel consensus broadcast pass ====")
	case <-passChan:
		fmt.Println("==== test cancel consensus broadcast failed ====")
		t.Fatal("test cancel consensus broadcast failed")
	}

	err = nsa.Stop()
	require.Nil(t, err)
	err = a.Stop()
	require.Nil(t, err)
	err = nsb.Stop()
	require.Nil(t, err)
	err = b.Stop()
	require.Nil(t, err)

}

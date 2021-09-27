/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

/*
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

// TestNetDataIsolate 网络侧多链数据隔离
func TestNetDataIsolate(t *testing.T) {
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
	caBytes8888, err := ioutil.ReadFile(filepath.Join(certPath, "ca3.crt"))
	require.Nil(t, err)
	key1Path := filepath.Join(certPath, "key1.key")
	cert1Path := filepath.Join(certPath, "cert1.crt")
	key2Path := filepath.Join(certPath, "key2.key")
	cert2Path := filepath.Join(certPath, "cert2.crt")
	pid2Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "pid2.nodeid"))
	require.Nil(t, err)
	pid2 := string(pid2Bytes)
	key3Path := filepath.Join(certPath, "key3.key")
	cert3Path := filepath.Join(certPath, "cert3.crt")
	pid3Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "pid3.nodeid"))
	require.Nil(t, err)
	pid3 := string(pid3Bytes)

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
	err = a.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + pid2)
	a.SetChainCustomTrustRoots(chainId1, [][]byte{caBytes6666, caBytes7777})
	err = a.Start()
	require.Nil(t, err)
	err = a.InitPubSub(chainId1, 0)
	require.Nil(t, err)
	fmt.Println("node A is running...")

	// start node B
	b, err := nf.NewNet(
		protocol.Libp2p,
		WithReadySignalC(readyC),
		WithListenAddr("/ip4/127.0.0.1/tcp/7777"),
		WithCrypto(false, key2Path, cert2Path),
	)
	require.Nil(t, err)
	b.SetChainCustomTrustRoots(chainId1, [][]byte{caBytes6666, caBytes7777})
	b.SetChainCustomTrustRoots(chainId2, [][]byte{caBytes8888, caBytes7777})
	err = b.Start()
	require.Nil(t, err)
	err = b.InitPubSub(chainId1, 0)
	require.Nil(t, err)
	err = b.InitPubSub(chainId2, 0)
	require.Nil(t, err)
	fmt.Println("node B is running...")
	// start node C
	c, err := nf.NewNet(
		protocol.Libp2p,
		WithReadySignalC(readyC),
		WithListenAddr("/ip4/127.0.0.1/tcp/8888"),
		WithCrypto(false, key3Path, cert3Path),
	)
	require.Nil(t, err)
	err = c.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + pid2)
	require.Nil(t, err)
	c.SetChainCustomTrustRoots(chainId2, [][]byte{caBytes8888, caBytes7777})
	err = c.Start()
	require.Nil(t, err)
	err = c.InitPubSub(chainId1, 0)
	require.Nil(t, err)
	err = c.InitPubSub(chainId2, 0)
	require.Nil(t, err)
	fmt.Println("node C is running...")

	close(readyC)

	// test A send msg to B
	data := []byte("hello")
	toNodeB := pid2
	toNodeC := pid3
	passChan := make(chan bool)

	aSendMsgToB(t, a, b, passChan, data, toNodeB)

	// test A send msg to C
	aSendMsgToC(t, a, c, passChan, data, toNodeC)

	// test A broadcast msg for chain1
	aBroadcastMsgToChain1(t, a, b, c, passChan, data)

	err = a.Stop()
	require.Nil(t, err)
	err = b.Stop()
	require.Nil(t, err)
	err = c.Stop()
	require.Nil(t, err)
}

func aSendMsgToB(t *testing.T, a, b protocol.Net, passChan chan bool, data []byte, toNodeB string) {
	recHandlerB := func(id string, msg []byte) error {
		fmt.Println("[B][chain1] recv a msg from peer[", id, "], msg：", string(msg))
		passChan <- true
		return nil
	}
	err := b.DirectMsgHandle(chainId1, msgFlag, recHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B register receive msg handler for chain1")
	go func() {
		fmt.Println("[A]A send msg to B in chain1")
		for {
			if err := a.SendMsg(chainId1, toNodeB, msgFlag, data); err != nil {
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
}

func aSendMsgToC(t *testing.T, a, c protocol.Net, passChan chan bool, data []byte, toNodeC string) {
	recHandlerC := func(id string, msg []byte) error {
		fmt.Println("[C][chain2] recv a msg from peer[", id, "], msg：", string(msg))
		passChan <- false
		return nil
	}
	err := c.DirectMsgHandle(chainId1, msgFlag, recHandlerC)
	require.Nil(t, err)
	fmt.Println("[C]C register receive msg handler for chain1")
	breakFlag := false
	go func() {
		fmt.Println("[A]A send msg to C in chain2")
		for {
			if breakFlag {
				break
			}
			if err := a.SendMsg(chainId2, toNodeC, msgFlag, data); err != nil {
				fmt.Println(err)
				time.Sleep(2 * time.Second)
				continue
			}
			break
		}
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("==== test A send msg to C : pass ====")
		breakFlag = true
	case <-passChan:
		fmt.Println("==== test A send msg to C success,BUG: data not isolate ====")
		t.Fatal("test A send msg to C success,BUG: data not isolate")
	}
}

func aBroadcastMsgToChain1(t *testing.T, a, b, c protocol.Net, passChan chan bool, data []byte) {
	testTopic := "testTopic"
	isBRecvSub := false
	subHandlerB := func(_ string, msg []byte) error {
		fmt.Println("[B]recv a sub msg of chain1：", string(msg))
		isBRecvSub = true
		return nil
	}
	err := b.SubscribeWithChainId(chainId1, testTopic, subHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B subscribe topic of chain1")

	subHandlerC := func(_ string, msg []byte) error {
		fmt.Println("[C]recv a sub msg of chain1：", string(msg))
		passChan <- false
		return nil
	}
	err = c.SubscribeWithChainId(chainId1, testTopic, subHandlerC)
	require.Nil(t, err)
	fmt.Println("[C]C subscribe topic of chain1")
	if err := a.BroadcastWithChainId(chainId1, testTopic, data); err != nil {
		t.Fatal(err)
	}
	fmt.Println("[A]chain1广播消息")

	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		if isBRecvSub {
			fmt.Println("==== test A broadcast msg for chain1 : pass ====")
		} else {
			fmt.Println("==== test A broadcast msg for chain1 fail,BUG: B do not recv sub msg ====")
			t.Fatal("test A broadcast msg for chain1 fail,BUG: B do not recv sub msg")
		}
	case <-passChan:
		fmt.Println("==== test A broadcast msg for chain1 fail,BUG: data not isolate ====")
		t.Fatal("test A send msg to C success,BUG: data not isolate")
	}
}*/

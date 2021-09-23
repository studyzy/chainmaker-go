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

// TestNetGmTls 网络侧国密TLS证书测试
func TestNetGmTls(t *testing.T) {
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
	caBytes6666, err := ioutil.ReadFile(filepath.Join(certPath, "gmca1.crt"))
	require.Nil(t, err)
	caBytes7777, err := ioutil.ReadFile(filepath.Join(certPath, "gmca2.crt"))
	require.Nil(t, err)
	key1Path := filepath.Join(certPath, "gmkey1.key")
	cert1Path := filepath.Join(certPath, "gmcert1.crt")
	pid1Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "gmpid1.nodeid"))
	require.Nil(t, err)
	pid1 := string(pid1Bytes)
	key2Path := filepath.Join(certPath, "gmkey2.key")
	cert2Path := filepath.Join(certPath, "gmcert2.crt")
	pid2Bytes, err := ioutil.ReadFile(filepath.Join(pidPath, "gmpid2.nodeid"))
	require.Nil(t, err)
	pid2 := string(pid2Bytes)

	// start node A
	var nf NetFactory

	a, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/6666"),
		WithCrypto(false, key1Path, cert1Path),
	)
	require.Nil(t, err)
	//a.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + pid2)
	a.SetChainCustomTrustRoots(chainId1, [][]byte{caBytes6666, caBytes7777})
	err = a.Start()
	require.Nil(t, err)
	fmt.Println("node A is running...")

	// start node B
	b, err := nf.NewNet(
		protocol.Libp2p,
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

	// test A send msg to B
	data := []byte("hello")
	toNodeB := pid2
	passChan := make(chan bool)
	recHandlerB := func(id string, msg []byte) error {
		fmt.Println("[B][chain1] recv a msg from peer[", id, "], msg：", string(msg))
		passChan <- true
		return nil
	}
	err = b.DirectMsgHandle(chainId1, msgFlag, recHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B register receive msg handler for chain1")
	go func() {
		fmt.Println("[A]A send msg to B in chain1")
		for {
			if err = a.SendMsg(chainId1, toNodeB, msgFlag, data); err != nil {
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

	err = a.Stop()
	require.Nil(t, err)
	err = b.Stop()
	require.Nil(t, err)
}*/

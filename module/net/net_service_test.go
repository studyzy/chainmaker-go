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
	"testing"
	"time"

	netPb "chainmaker.org/chainmaker/pb-go/net"
	"chainmaker.org/chainmaker/protocol"
	"github.com/stretchr/testify/require"
)

func TestNetService(t *testing.T) {
	var td = filepath.Join(os.TempDir(), "temp")
	err := os.MkdirAll(td, os.ModePerm)
	require.Nil(t, err)
	defer func() {
		_ = os.RemoveAll(td)
		_ = os.RemoveAll(filepath.Join("./default.log"))
		now := time.Now()
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
		now = now.Add(-5 * time.Hour)
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
	}()
	key6666 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF4Sy4KANZHi8uU4YkmymbcbF3HHJnGgSjV/0iNOSdy3oAoGCCqGSM49\nAwEHoUQDQgAEKwemRhrzv5GSSmsy4EREhnQJ4jocauyWnD1dXUx9X8c4VwhG5hWQ\n7oc+cMyz6rXPKTrUxKD50V+OB0FVkpY7vA==\n-----END EC PRIVATE KEY-----\n"
	cert6666 := "-----BEGIN CERTIFICATE-----\nMIIDFTCCArugAwIBAgIDBOOCMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzEuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQr\nB6ZGGvO/kZJKazLgRESGdAniOhxq7JacPV1dTH1fxzhXCEbmFZDuhz5wzLPqtc8p\nOtTEoPnRX44HQVWSlju8o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgqzFBKQ6cAvTThFgrn//B/SDhAFEDfW5Y8MOE7hvY\nBf4wKwYDVR0jBCQwIoAgNSQ/cRy5t8Q1LpMfcMVzMfl0CcLZ4Pvf7BxQX9sQiWcw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMS5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgMDAxNjQ2ZTY3ODBmNGIwZDhiZWEzMjNlZThjMjQ5MTUwCgYIKoZIzj0EAwID\nSAAwRQIgNVNGr+G8dbYnzmmNMr9GCSUEC3TUmRcS4uOd5/Sw4mECIQDII1R7dCcx\n02YrxI8jEQZhmWeZ5FJhnSG6p6H9pCIWDQ==\n-----END CERTIFICATE-----\n"
	ca6666 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n-----END CERTIFICATE-----\n"
	key7777 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIimV5TA1i8QWlp5nD5r5KmpueJV1hplp5y7Of4CYquzoAoGCCqGSM49\nAwEHoUQDQgAESZXYY4gziokaliXX5JkwT+idTCCwesjuJtTupABuhIqu7o2jt1V0\nNNWVvpShIM+878BaSb2v2TllwVoOYmfzPg==\n-----END EC PRIVATE KEY-----\n"
	cert7777 := "-----BEGIN CERTIFICATE-----\nMIIDFjCCArugAwIBAgIDAdGZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzIuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARJ\nldhjiDOKiRqWJdfkmTBP6J1MILB6yO4m1O6kAG6Eiq7ujaO3VXQ01ZW+lKEgz7zv\nwFpJva/ZOWXBWg5iZ/M+o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgH0PY7Oic1NRq5O64ag3g12d5vI5jqEWW9+MzOOrE\nnhEwKwYDVR0jBCQwIoAg8Y/Vs9Pj8uezY+di51n3+oexybSkYvop/L7UIAVYbSEw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMi5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgZjVhODUwYTAzYjFlNDU0NzkzOTg5NzIxYzVjMTc3NjMwCgYIKoZIzj0EAwID\nSQAwRgIhAKvDGBl+17dcTMdOjRW3VTTaGNlQiZepRXYarmAdX3PiAiEA6F6cZjsT\nEpSBfal9mUGlxJNNHhYIxs2SlSL4of4GTBA=\n-----END CERTIFICATE-----\n"
	ca7777 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDYpTMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASlekil12ThyvibHhBn\ncDvu958HOdN5Db9YE8bZ5e7YYHsJ85P6jBhlt0eKTR/hiukIBVfYKYwmhpYq2eCb\nRYqco4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIPGP1bPT4/Lns2PnYudZ9/qHscm0pGL6Kfy+\n1CAFWG0hMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzIuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nJV7mg6IeKBVSLrsDFpLOSEMFd9zKIxo3RRZiMAkdC3MCIQD/LG53Sb/IcNsCqjz9\noLXYNanXzZn1c1t4jPtMuE7nSw==\n-----END CERTIFICATE-----\n"
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.key"), []byte(key6666), 0777))
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.crt"), []byte(cert6666), 0777))
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.ca.crt"), []byte(ca6666), 0777))
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.key"), []byte(key7777), 0777))
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.crt"), []byte(cert7777), 0777))
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.ca.crt"), []byte(ca7777), 0777))
	caBytes6666, err := ioutil.ReadFile(filepath.Join(td, "6666.ca.crt"))
	require.Nil(t, err)
	caBytes7777, err := ioutil.ReadFile(filepath.Join(td, "7777.ca.crt"))
	require.Nil(t, err)

	// start node A
	var nf NetFactory

	a, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/6666"),

		WithCrypto(filepath.Join(td, "6666.key"), filepath.Join(td, "6666.crt")),
	)
	require.Nil(t, err)
	//a.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH")
	err = a.AddTrustRoot(chainId1, caBytes6666)
	require.Nil(t, err)
	err = a.AddTrustRoot(chainId1, caBytes7777)
	require.Nil(t, err)

	err = a.Start()
	require.Nil(t, err)
	fmt.Println("node A is running...")

	var nsf NetServiceFactory
	nsa, err := nsf.NewNetService(
		a,
		chainId1,
		nil,
		nil,
		WithConsensusNodeUid("QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"),
	)
	require.Nil(t, err)
	err = nsa.Start()
	require.Nil(t, err)
	fmt.Println("net service A is running...")
	// start node B
	b, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/7777"),

		WithCrypto(filepath.Join(td, "7777.key"), filepath.Join(td, "7777.crt")),
	)
	require.Nil(t, err)
	err = b.AddSeed("/ip4/127.0.0.1/tcp/6666/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4")
	require.Nil(t, err)
	err = b.AddTrustRoot(chainId1, caBytes6666)
	require.Nil(t, err)
	err = b.AddTrustRoot(chainId1, caBytes7777)
	require.Nil(t, err)
	err = b.Start()
	require.Nil(t, err)
	fmt.Println("node B is running...")

	nsb, err := nsf.NewNetService(
		b,
		chainId1,
		nil,
		nil,
		WithConsensusNodeUid("QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4"),
	)
	require.Nil(t, err)
	err = nsb.Start()
	require.Nil(t, err)
	fmt.Println("net service B is running...")

	// test A send msg to B
	data := []byte("hello")
	toNodeB := "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"
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
		for {
			if err = nsa.SendMsg(data, netPb.NetMsg_TX, toNodeB); err != nil {
				fmt.Println(err)
				time.Sleep(2 * time.Second)
				continue
			}
			fmt.Println("[A]A send msg to B in chain1")
			break
		}
	}()
	timer := time.NewTimer(15 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("==== test A send msg to B timeout ====")
		t.Fatal("test A send msg to B timeout")
	case <-passChan:
		fmt.Println("==== test A send msg to B pass ====")
	}

	// test broadcast
	subHandlerB := func(_ string, msg []byte, _ netPb.NetMsg_MsgType) error {
		fmt.Println("[B][chain1] recv a sub msg chain1：", string(msg))
		passChan <- true
		return nil
	}
	err = nsb.Subscribe(netPb.NetMsg_TX, subHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B subscribe topic of chain1")

	err = nsa.BroadcastMsg(data, netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[A]A broadcast a msg to chain1:", string(data))

	timer = time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("==== test broadcast timeout ====")
		t.Fatal("test broadcast failed")
	case <-passChan:
		fmt.Println("==== test broadcast pass ====")
	}

	// test cancel broadcast
	err = nsb.CancelSubscribe(netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[B]B cancel subscribe topic of chain1")

	err = nsa.BroadcastMsg(data, netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[A]A broadcast a msg to chain1:", string(data))

	timer = time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("==== test cancel broadcast pass ====")
	case <-passChan:
		fmt.Println("==== test cancel broadcast failed ====")
		t.Fatal("test cancel broadcast failed")
	}

	// test consensus broadcast
	err = nsb.ConsensusSubscribe(netPb.NetMsg_TX, subHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B subscribe consensus topic of chain1")

	err = nsa.ConsensusBroadcastMsg(data, netPb.NetMsg_TX)
	require.Nil(t, err)
	fmt.Println("[A]A broadcast a msg to chain1 consensus nodes:", string(data))

	timer = time.NewTimer(10 * time.Second)
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

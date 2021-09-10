/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	netPb "chainmaker.org/chainmaker/pb-go/v2/net"

	"chainmaker.org/chainmaker-go/net"
	"chainmaker.org/chainmaker/protocol/v2"
)

func main() {
	var bindIp, otherIp string
	var c, streamPoolSize, msgKb int
	flag.IntVar(&c, "c", -1, "which client want to use: 1 sender; 2 other")
	flag.IntVar(&streamPoolSize, "stream-pool", 500, "sender stream pool size")
	flag.IntVar(&msgKb, "msg-kb", 5120, "message size (kb)")
	flag.StringVar(&bindIp, "bind", "127.0.0.1", "client bind ip")
	flag.StringVar(&otherIp, "other", "127.0.0.1", "otherIp bind ip")
	flag.Parse()
	switch c {
	case 1:
		err := StartSender(bindIp, otherIp, msgKb)
		if err != nil {
			fmt.Printf("start sender err,%s \n", err.Error())
		}
	case 2:
		err := StartReceiver(otherIp, bindIp)
		if err != nil {
			fmt.Printf("start receiver err,%s \n", err.Error())
		}
	default:
		fmt.Println("unknown client")
	}
}

var (
	chain1           = "chain1"
	chain2           = "chain2"
	key6666Name      = "6666.key"
	crt6666Name      = "6666.crt"
	cnetConn6666Name = "6666.cnetConn.crt"
	key7777Name      = "7777.key"
	crt7777Name      = "7777.crt"
	cnetConn7777Name = "7777.cnetConn.crt"
)

func StartSender(senderIp, receiverIp string, msgKb int) error {
	var td = filepath.Join(os.TempDir(), "chainmaker_net_test_key")
	os.Mkdir(td, 0666)
	key6666 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF4Sy4KANZHi8uU4YkmymbcbF3HHJnGgSjV/0iNOSdy3oAoGCCqGSM49\nAwEHoUQDQgAEKwemRhrzv5GSSmsy4EREhnQJ4jocauyWnD1dXUx9X8c4VwhG5hWQ\n7oc+cMyz6rXPKTrUxKD50V+OB0FVkpY7vA==\n-----END EC PRIVATE KEY-----\n"
	cert6666 := "-----BEGIN CERTIFICATE-----\nMIIDFTCCArugAwIBAgIDBOOCMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzEuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQr\nB6ZGGvO/kZJKazLgRESGdAniOhxq7JacPV1dTH1fxzhXCEbmFZDuhz5wzLPqtc8p\nOtTEoPnRX44HQVWSlju8o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgqzFBKQ6cAvTThFgrn//B/SDhAFEDfW5Y8MOE7hvY\nBf4wKwYDVR0jBCQwIoAgNSQ/cRy5t8Q1LpMfcMVzMfl0CcLZ4Pvf7BxQX9sQiWcw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMS5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgMDAxNjQ2ZTY3ODBmNGIwZDhiZWEzMjNlZThjMjQ5MTUwCgYIKoZIzj0EAwID\nSAAwRQIgNVNGr+G8dbYnzmmNMr9GCSUEC3TUmRcS4uOd5/Sw4mECIQDII1R7dCcx\n02YrxI8jEQZhmWeZ5FJhnSG6p6H9pCIWDQ==\n-----END CERTIFICATE-----\n"
	ca6666 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n-----END CERTIFICATE-----\n"
	key7777 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIimV5TA1i8QWlp5nD5r5KmpueJV1hplp5y7Of4CYquzoAoGCCqGSM49\nAwEHoUQDQgAESZXYY4gziokaliXX5JkwT+idTCCwesjuJtTupABuhIqu7o2jt1V0\nNNWVvpShIM+878BaSb2v2TllwVoOYmfzPg==\n-----END EC PRIVATE KEY-----\n"
	cert7777 := "-----BEGIN CERTIFICATE-----\nMIIDFjCCArugAwIBAgIDAdGZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzIuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARJ\nldhjiDOKiRqWJdfkmTBP6J1MILB6yO4m1O6kAG6Eiq7ujaO3VXQ01ZW+lKEgz7zv\nwFpJva/ZOWXBWg5iZ/M+o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgH0PY7Oic1NRq5O64ag3g12d5vI5jqEWW9+MzOOrE\nnhEwKwYDVR0jBCQwIoAg8Y/Vs9Pj8uezY+di51n3+oexybSkYvop/L7UIAVYbSEw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMi5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgZjVhODUwYTAzYjFlNDU0NzkzOTg5NzIxYzVjMTc3NjMwCgYIKoZIzj0EAwID\nSQAwRgIhAKvDGBl+17dcTMdOjRW3VTTaGNlQiZepRXYarmAdX3PiAiEA6F6cZjsT\nEpSBfal9mUGlxJNNHhYIxs2SlSL4of4GTBA=\n-----END CERTIFICATE-----\n"
	ca7777 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDYpTMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASlekil12ThyvibHhBn\ncDvu958HOdN5Db9YE8bZ5e7YYHsJ85P6jBhlt0eKTR/hiukIBVfYKYwmhpYq2eCb\nRYqco4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIPGP1bPT4/Lns2PnYudZ9/qHscm0pGL6Kfy+\n1CAFWG0hMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzIuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nJV7mg6IeKBVSLrsDFpLOSEMFd9zKIxo3RRZiMAkdC3MCIQD/LG53Sb/IcNsCqjz9\noLXYNanXzZn1c1t4jPtMuE7nSw==\n-----END CERTIFICATE-----\n"
	ioutil.WriteFile(filepath.Join(td, key6666Name), []byte(key6666), 0600)
	ioutil.WriteFile(filepath.Join(td, key6666Name), []byte(cert6666), 0666)
	ioutil.WriteFile(filepath.Join(td, cnetConn6666Name), []byte(ca6666), 0666)
	ioutil.WriteFile(filepath.Join(td, key7777Name), []byte(key7777), 0600)
	ioutil.WriteFile(filepath.Join(td, crt7777Name), []byte(cert7777), 0666)
	ioutil.WriteFile(filepath.Join(td, cnetConn7777Name), []byte(ca7777), 0666)
	caBytes6666, _ := ioutil.ReadFile(filepath.Join(td, cnetConn6666Name))
	caBytes7777, _ := ioutil.ReadFile(filepath.Join(td, cnetConn7777Name))

	// start node A
	var (
		nf      net.NetFactory
		netConn net.Net
		err     error
	)

	if netConn, err = nf.NewNet(
		protocol.Libp2p,
		net.WithListenAddr(fmt.Sprintf("/ip4/%s/tcp/6666", senderIp)),
		net.WithCrypto(filepath.Join(td, key6666Name), filepath.Join(td, key6666Name)),
	); err != nil {
		return err
	}
	netConn.AddSeed(fmt.Sprintf("/ip4/%s/tcp/7777/p2p/QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH", receiverIp))
	netConn.AddTrustRoot(chain1, caBytes6666)
	netConn.AddTrustRoot(chain1, caBytes7777)
	netConn.InitPubsub(chain1, 0)
	netConn.AddTrustRoot(chain2, caBytes6666)
	netConn.AddTrustRoot(chain2, caBytes7777)
	netConn.InitPubsub(chain2, 0)
	if err = netConn.Start(); err != nil {
		return err
	}
	fmt.Println("node A is running...")

	// test A send msg to B
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(1)
	}
	data2 := make([]byte, 0)
	for i := 0; i < msgKb; i++ {
		data2 = append(data2, data...)
	}
	fmt.Println("data length: ", len(data2))
	toNodeB := "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"
	sendMsg := net.NewNetMsg(data2, netPb.NetMsg_TX, toNodeB)
	count := 0
	lock := sync.Mutex{}

	go sender(netConn, toNodeB, sendMsg, &lock, &count)

	go timer(&lock, &count)

	select {}

}

func sender(netConn net.Net, to string, sendMsg *netPb.NetMsg, lock *sync.Mutex, count *int) {
	var err error
	for i := 0; i < 100; i++ {
		go func() {
			for {
				if err = netConn.SendMsg(chain1, to, "TEST_PUSH", sendMsg); err != nil {
					fmt.Println(err)
					time.Sleep(2 * time.Second)
					continue
				}
				go func() {
					lock.Lock()
					*count++
					lock.Unlock()
				}()
			}
		}()
	}
}

func timer(lock *sync.Mutex, count *int) {
	timer := time.NewTimer(time.Second)
	for {
		timer.Reset(time.Second)
		select {
		case <-timer.C:
			lock.Lock()
			fmt.Println("current ops:", count)
			*count = 0
			lock.Unlock()
		}
	}
}

func StartReceiver(senderIp, receiverIp string) error {
	var td = filepath.Join(os.TempDir(), "chainmaker_net_test_key")
	os.Mkdir(td, 0666)
	key6666 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF4Sy4KANZHi8uU4YkmymbcbF3HHJnGgSjV/0iNOSdy3oAoGCCqGSM49\nAwEHoUQDQgAEKwemRhrzv5GSSmsy4EREhnQJ4jocauyWnD1dXUx9X8c4VwhG5hWQ\n7oc+cMyz6rXPKTrUxKD50V+OB0FVkpY7vA==\n-----END EC PRIVATE KEY-----\n"
	cert6666 := "-----BEGIN CERTIFICATE-----\nMIIDFTCCArugAwIBAgIDBOOCMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzEuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQr\nB6ZGGvO/kZJKazLgRESGdAniOhxq7JacPV1dTH1fxzhXCEbmFZDuhz5wzLPqtc8p\nOtTEoPnRX44HQVWSlju8o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgqzFBKQ6cAvTThFgrn//B/SDhAFEDfW5Y8MOE7hvY\nBf4wKwYDVR0jBCQwIoAgNSQ/cRy5t8Q1LpMfcMVzMfl0CcLZ4Pvf7BxQX9sQiWcw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMS5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgMDAxNjQ2ZTY3ODBmNGIwZDhiZWEzMjNlZThjMjQ5MTUwCgYIKoZIzj0EAwID\nSAAwRQIgNVNGr+G8dbYnzmmNMr9GCSUEC3TUmRcS4uOd5/Sw4mECIQDII1R7dCcx\n02YrxI8jEQZhmWeZ5FJhnSG6p6H9pCIWDQ==\n-----END CERTIFICATE-----\n"
	ca6666 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n-----END CERTIFICATE-----\n"
	key7777 := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIimV5TA1i8QWlp5nD5r5KmpueJV1hplp5y7Of4CYquzoAoGCCqGSM49\nAwEHoUQDQgAESZXYY4gziokaliXX5JkwT+idTCCwesjuJtTupABuhIqu7o2jt1V0\nNNWVvpShIM+878BaSb2v2TllwVoOYmfzPg==\n-----END EC PRIVATE KEY-----\n"
	cert7777 := "-----BEGIN CERTIFICATE-----\nMIIDFjCCArugAwIBAgIDAdGZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzIuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARJ\nldhjiDOKiRqWJdfkmTBP6J1MILB6yO4m1O6kAG6Eiq7ujaO3VXQ01ZW+lKEgz7zv\nwFpJva/ZOWXBWg5iZ/M+o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgH0PY7Oic1NRq5O64ag3g12d5vI5jqEWW9+MzOOrE\nnhEwKwYDVR0jBCQwIoAg8Y/Vs9Pj8uezY+di51n3+oexybSkYvop/L7UIAVYbSEw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMi5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgZjVhODUwYTAzYjFlNDU0NzkzOTg5NzIxYzVjMTc3NjMwCgYIKoZIzj0EAwID\nSQAwRgIhAKvDGBl+17dcTMdOjRW3VTTaGNlQiZepRXYarmAdX3PiAiEA6F6cZjsT\nEpSBfal9mUGlxJNNHhYIxs2SlSL4of4GTBA=\n-----END CERTIFICATE-----\n"
	ca7777 := "-----BEGIN CERTIFICATE-----\nMIICrzCCAlWgAwIBAgIDDYpTMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASlekil12ThyvibHhBn\ncDvu958HOdN5Db9YE8bZ5e7YYHsJ85P6jBhlt0eKTR/hiukIBVfYKYwmhpYq2eCb\nRYqco4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIPGP1bPT4/Lns2PnYudZ9/qHscm0pGL6Kfy+\n1CAFWG0hMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzIuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nJV7mg6IeKBVSLrsDFpLOSEMFd9zKIxo3RRZiMAkdC3MCIQD/LG53Sb/IcNsCqjz9\noLXYNanXzZn1c1t4jPtMuE7nSw==\n-----END CERTIFICATE-----\n"
	ioutil.WriteFile(filepath.Join(td, key6666Name), []byte(key6666), 0600)
	ioutil.WriteFile(filepath.Join(td, key6666Name), []byte(cert6666), 0666)
	ioutil.WriteFile(filepath.Join(td, cnetConn6666Name), []byte(ca6666), 0666)
	ioutil.WriteFile(filepath.Join(td, key7777Name), []byte(key7777), 0600)
	ioutil.WriteFile(filepath.Join(td, crt7777Name), []byte(cert7777), 0666)
	ioutil.WriteFile(filepath.Join(td, cnetConn7777Name), []byte(ca7777), 0666)
	caBytes6666, _ := ioutil.ReadFile(filepath.Join(td, cnetConn6666Name))
	caBytes7777, _ := ioutil.ReadFile(filepath.Join(td, cnetConn7777Name))

	// start node A
	var nf net.NetFactory

	b, err := nf.NewNet(
		protocol.Libp2p,
		net.WithListenAddr(fmt.Sprintf("/ip4/%s/tcp/7777", receiverIp)),

		net.WithCrypto(filepath.Join(td, key7777Name), filepath.Join(td, crt7777Name)),
	)
	if err != nil {
		return err
	}
	b.AddSeed(fmt.Sprintf("/ip4/%s/tcp/6666/p2p/QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4", senderIp))
	b.AddTrustRoot(chain1, caBytes6666)
	b.AddTrustRoot(chain1, caBytes7777)
	b.InitPubsub(chain1, 0)
	b.AddTrustRoot(chain2, caBytes6666)
	b.AddTrustRoot(chain2, caBytes7777)
	b.InitPubsub(chain2, 0)
	err = b.Start()
	if err != nil {
		return nil
	}
	fmt.Println("node B is running...")

	count := 0
	lock := sync.Mutex{}

	recHandlerB := func(id string, msg *netPb.NetMsg) error {
		fmt.Println("[B][chain1] recv a msg from peer[", id, "], msg：", string(msg.GetPayload()))
		go func() {
			lock.Lock()
			count++
			lock.Unlock()
		}()
		return nil
	}

	if err = b.DirectMsgHandle(chain1, "TEST_PUSH", recHandlerB); err != nil {
		return err
	}

	go func() {
		timer := time.NewTimer(time.Second)
		for {
			timer.Reset(time.Second)
			select {
			case <-timer.C:
				lock.Lock()
				fmt.Println("current ops:", count)
				count = 0
				lock.Unlock()
			}
		}
	}()

	select {}
}

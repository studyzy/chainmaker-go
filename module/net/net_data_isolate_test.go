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

	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

// TestNetDataIsolate 网络侧多链数据隔离
func TestNetDataIsolate(t *testing.T) {
	var td = filepath.Join(os.TempDir(), "temp")
	require.Nil(t, os.MkdirAll(td, os.ModePerm))
	defer func() {
		_ = os.RemoveAll(td)
		_ = os.RemoveAll(filepath.Join("./default.log"))
		now := time.Now()
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
		now = now.Add(-5 * time.Hour)
		_ = os.RemoveAll(filepath.Join("./default.log." + now.Format("2006010215")))
	}()
	const (
		keyBegin = "-----BEGIN EC PRIVATE KEY-----"
		keyEnd   = "-----END EC PRIVATE KEY-----"
		crtBegin = "-----BEGIN CERTIFICATE-----"
		crtEnd   = "-----END CERTIFICATE-----"
		seed1    = "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"
		seed2    = "QmXf6mnQDBR9aHauRmViKzSuZgpumkn7x6rNxw1oqqRr45"
	)
	key6666 := keyBegin + "\nMHcCAQEEIF4Sy4KANZHi8uU4YkmymbcbF3HHJnGgSjV/0iNOSdy3oAoGCCqGSM49\nAwEHoUQDQgAEKwemRhrzv5GSSmsy4EREhnQJ4jocauyWnD1dXUx9X8c4VwhG5hWQ\n7oc+cMyz6rXPKTrUxKD50V+OB0FVkpY7vA==\n" + keyEnd + "\n"
	cert6666 := crtBegin + "\nMIIDFTCCArugAwIBAgIDBOOCMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzEuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQr\nB6ZGGvO/kZJKazLgRESGdAniOhxq7JacPV1dTH1fxzhXCEbmFZDuhz5wzLPqtc8p\nOtTEoPnRX44HQVWSlju8o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgqzFBKQ6cAvTThFgrn//B/SDhAFEDfW5Y8MOE7hvY\nBf4wKwYDVR0jBCQwIoAgNSQ/cRy5t8Q1LpMfcMVzMfl0CcLZ4Pvf7BxQX9sQiWcw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMS5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgMDAxNjQ2ZTY3ODBmNGIwZDhiZWEzMjNlZThjMjQ5MTUwCgYIKoZIzj0EAwID\nSAAwRQIgNVNGr+G8dbYnzmmNMr9GCSUEC3TUmRcS4uOd5/Sw4mECIQDII1R7dCcx\n02YrxI8jEQZhmWeZ5FJhnSG6p6H9pCIWDQ==\n" + crtEnd + "\n"
	ca6666 := crtBegin + "\nMIICrzCCAlWgAwIBAgIDDsPeMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAT7NyTIKcjtUVeMn29b\nGKeEmwbefZ7g9Uk5GROl+o4k7fiIKNuty1rQHLQUvAvkpxqtlmOpPOZ0Qziu6Hw6\nhi19o4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wc\nUF/bEIlnMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzEuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nar8CSuLl7pA4Iy6ytAMhR0kzy0WWVSElc+koVY6pF5sCIQCDs+vTD/9V1azmbDXX\nbjoWeEfXbFJp2X/or9f4UIvMgg==\n" + crtEnd + "\n"
	key7777 := keyBegin + "\nMHcCAQEEIIimV5TA1i8QWlp5nD5r5KmpueJV1hplp5y7Of4CYquzoAoGCCqGSM49\nAwEHoUQDQgAESZXYY4gziokaliXX5JkwT+idTCCwesjuJtTupABuhIqu7o2jt1V0\nNNWVvpShIM+878BaSb2v2TllwVoOYmfzPg==\n" + keyEnd + "\n"
	cert7777 := crtBegin + "\nMIIDFjCCArugAwIBAgIDAdGZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzIuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARJ\nldhjiDOKiRqWJdfkmTBP6J1MILB6yO4m1O6kAG6Eiq7ujaO3VXQ01ZW+lKEgz7zv\nwFpJva/ZOWXBWg5iZ/M+o4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgH0PY7Oic1NRq5O64ag3g12d5vI5jqEWW9+MzOOrE\nnhEwKwYDVR0jBCQwIoAg8Y/Vs9Pj8uezY+di51n3+oexybSkYvop/L7UIAVYbSEw\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMi5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgZjVhODUwYTAzYjFlNDU0NzkzOTg5NzIxYzVjMTc3NjMwCgYIKoZIzj0EAwID\nSQAwRgIhAKvDGBl+17dcTMdOjRW3VTTaGNlQiZepRXYarmAdX3PiAiEA6F6cZjsT\nEpSBfal9mUGlxJNNHhYIxs2SlSL4of4GTBA=\n" + crtEnd + "\n"
	ca7777 := crtBegin + "\nMIICrzCCAlWgAwIBAgIDDYpTMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASlekil12ThyvibHhBn\ncDvu958HOdN5Db9YE8bZ5e7YYHsJ85P6jBhlt0eKTR/hiukIBVfYKYwmhpYq2eCb\nRYqco4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEIPGP1bPT4/Lns2PnYudZ9/qHscm0pGL6Kfy+\n1CAFWG0hMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzIuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nJV7mg6IeKBVSLrsDFpLOSEMFd9zKIxo3RRZiMAkdC3MCIQD/LG53Sb/IcNsCqjz9\noLXYNanXzZn1c1t4jPtMuE7nSw==\n" + crtEnd + "\n"
	key8888 := keyBegin + "\nMHcCAQEEIBOdWWD5V7dgz/q9PaQ3lyXddMuscws80fnI8Spo0PFYoAoGCCqGSM49\nAwEHoUQDQgAELUQWoVacLUfxlCHIc3OaosHj0MnnwV61i6z9ltBHLGB3vltuW29V\nt+vTgK2QregXLQUyzsS/w5dlpPyjwbMyrg==\n" + keyEnd + "\n"
	cert8888 := crtBegin + "\nMIIDFTCCArugAwIBAgIDCJoJMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwljb25zZW5zdXMxLjAsBgNVBAMTJWNvbnNlbnN1czEudGxzLnd4\nLW9yZzMuY2hhaW5tYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQt\nRBahVpwtR/GUIchzc5qiwePQyefBXrWLrP2W0EcsYHe+W25bb1W369OArZCt6Bct\nBTLOxL/Dl2Wk/KPBszKuo4IBADCB/TAOBgNVHQ8BAf8EBAMCAaYwDwYDVR0lBAgw\nBgYEVR0lADApBgNVHQ4EIgQgEnC2getHs64R4n9VVe1A66N41/5HH63o63aV8Iqq\nk2EwKwYDVR0jBCQwIoAg0Y9lHSxXCu9i0Wd5MPoZTIFB+XClOYnSoKyC90WAif0w\nUQYDVR0RBEowSIIOY2hhaW5tYWtlci5vcmeCCWxvY2FsaG9zdIIlY29uc2Vuc3Vz\nMS50bHMud3gtb3JnMy5jaGFpbm1ha2VyLm9yZ4cEfwAAATAvBguBJ1iPZAsej2QL\nBAQgNzNiMWM4MWJkZjA2NDllNjk4YmI4MTVlNWI3NzM2YmIwCgYIKoZIzj0EAwID\nSAAwRQIhAODEcNO5jIBT+Dd4Fcsxz1ML8pzIzcWlPDeeuD6nfbQMAiARIw6KvJMu\nH9A4TrVomaX3eP0ttXTYwhdqu+5JeA+j2Q==\n" + crtEnd + "\n"
	ca8888 := crtBegin + "\nMIICrzCCAlWgAwIBAgIDDjhZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMy5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmczLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw\nMTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmczLmNoYWlubWFrZXIub3Jn\nMRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzMuY2hhaW5t\nYWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAREQ8bC/Ocg6Nf1c0OG\nQXybPYWXT0fWygGvn2KgrBFQjq8NLOwXQPO4BYY1vYuBTTFl0Qf0uz7OvVPMcrmy\n6ZDXo4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud\nEwEB/wQFMAMBAf8wKQYDVR0OBCIEINGPZR0sVwrvYtFneTD6GUyBQflwpTmJ0qCs\ngvdFgIn9MEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh\nLnd4LW9yZzMuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg\nOxwZGMwSa58xWiou+Bpi6YcerIwm7Lqsd+4OqjHZp8ACIQCGElUBWJt5EYKkxt3x\nNb1ypMnQXHMFaHZVOIACtGz2GA==\n" + crtEnd + "\n"
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.key"), []byte(key6666), 0777))   //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.crt"), []byte(cert6666), 0777))  //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "6666.ca.crt"), []byte(ca6666), 0777)) //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.key"), []byte(key7777), 0777))   //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.crt"), []byte(cert7777), 0777))  //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "7777.ca.crt"), []byte(ca7777), 0777)) //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "8888.key"), []byte(key8888), 0777))   //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "8888.crt"), []byte(cert8888), 0777))  //nolint:gosec
	require.Nil(t, ioutil.WriteFile(filepath.Join(td, "8888.ca.crt"), []byte(ca8888), 0777)) //nolint:gosec
	caBytes6666, err := ioutil.ReadFile(filepath.Join(td, "6666.ca.crt"))
	require.Nil(t, err)
	caBytes7777, err := ioutil.ReadFile(filepath.Join(td, "7777.ca.crt"))
	require.Nil(t, err)
	caBytes8888, err := ioutil.ReadFile(filepath.Join(td, "8888.ca.crt"))
	require.Nil(t, err)

	// start node A
	var nf NetFactory

	a, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/6666"),
		WithCrypto(filepath.Join(td, "6666.key"), filepath.Join(td, "6666.crt")),
	)
	require.Nil(t, err)
	err = a.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + seed1)
	err = a.AddTrustRoot(chainId1, caBytes6666)
	err = a.AddTrustRoot(chainId1, caBytes7777)
	require.Nil(t, a.InitPubsub(chainId1, 0))
	require.Nil(t, a.Start())

	fmt.Println("node A is running...")

	// start node B
	b, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/7777"),
		WithCrypto(filepath.Join(td, "7777.key"), filepath.Join(td, "7777.crt")),
	)
	require.Nil(t, err)
	require.Nil(t, b.AddTrustRoot(chainId1, caBytes6666))
	require.Nil(t, b.AddTrustRoot(chainId1, caBytes7777))
	require.Nil(t, b.AddTrustRoot(chainId2, caBytes8888))
	require.Nil(t, b.AddTrustRoot(chainId2, caBytes7777))
	require.Nil(t, b.InitPubsub(chainId1, 0))
	require.Nil(t, b.InitPubsub(chainId2, 0))
	require.Nil(t, b.Start())
	fmt.Println("node B is running...")
	// start node C
	c, err := nf.NewNet(
		protocol.Libp2p,
		WithListenAddr("/ip4/127.0.0.1/tcp/8888"),
		WithCrypto(filepath.Join(td, "8888.key"), filepath.Join(td, "8888.crt")),
	)
	require.Nil(t, err)
	err = c.AddSeed("/ip4/127.0.0.1/tcp/7777/p2p/" + seed1)
	require.Nil(t, err)
	err = c.AddTrustRoot(chainId2, caBytes8888)
	require.Nil(t, err)
	err = c.AddTrustRoot(chainId2, caBytes7777)
	require.Nil(t, err)
	err = c.InitPubsub(chainId1, 0)
	require.Nil(t, err)
	err = c.InitPubsub(chainId2, 0)
	require.Nil(t, err)
	require.Nil(t, c.Start())
	fmt.Println("node C is running...")
	// test A send msg to B
	data := []byte("hello")
	toNodeB := seed1
	toNodeC := seed2
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

func aSendMsgToB(t *testing.T, a, b Net, passChan chan bool, data []byte, toNodeB string) {
	recHandlerB := func(id string, msg *netPb.NetMsg) error {
		fmt.Println("[B][chain1] recv a msg from peer[", id, "], msg：", string(msg.GetPayload()))
		passChan <- true
		return nil
	}
	err := b.DirectMsgHandle(chainId1, msgFlag, recHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B register receive msg handler for chain1")
	go func() {
		fmt.Println("[A]A send msg to B in chain1")
		for {
			sendMsg := NewNetMsg(data, netPb.NetMsg_TX, toNodeB)
			if err := a.SendMsg(chainId1, toNodeB, msgFlag, sendMsg); err != nil {
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

func aSendMsgToC(t *testing.T, a, c Net, passChan chan bool, data []byte, toNodeC string) {
	recHandlerC := func(id string, msg *netPb.NetMsg) error {
		fmt.Println("[C][chain2] recv a msg from peer[", id, "], msg：", string(msg.GetPayload()))
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
			sendMsg := NewNetMsg(data, netPb.NetMsg_TX, toNodeC)
			if err := a.SendMsg(chainId2, toNodeC, msgFlag, sendMsg); err != nil {
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

func aBroadcastMsgToChain1(t *testing.T, a, b, c Net, passChan chan bool, data []byte) {
	testTopic := "testTopic" //nolint: goconst
	isBRecvSub := false
	subHandlerB := func(_ string, msg *netPb.NetMsg) error {
		fmt.Println("[B]recv a sub msg of chain1：", string(msg.GetPayload()))
		isBRecvSub = true
		return nil
	}
	err := b.SubscribeWithChainId(chainId1, testTopic, subHandlerB)
	require.Nil(t, err)
	fmt.Println("[B]B subscribe topic of chain1")

	subHandlerC := func(_ string, msg *netPb.NetMsg) error {
		fmt.Println("[C]recv a sub msg of chain1：", string(msg.GetPayload()))
		passChan <- false
		return nil
	}
	err = c.SubscribeWithChainId(chainId1, testTopic, subHandlerC)
	require.Nil(t, err)
	fmt.Println("[C]C subscribe topic of chain1")
	sendMsg := NewNetMsg(data, netPb.NetMsg_TX, "")
	if err := a.BroadcastWithChainId(chainId1, testTopic, sendMsg); err != nil {
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
}

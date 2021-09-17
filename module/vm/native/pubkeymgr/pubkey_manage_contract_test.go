/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package pubkeymgr

import (
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"testing"
)

const KeyFormat = "%s-%s"

func realKey(name, key string) string {
	return fmt.Sprintf(KeyFormat, name, key)
}

type CacheMock struct {
	content map[string][]byte
}

func NewCacheMock() *CacheMock {
	return &CacheMock{
		content: make(map[string][]byte, 64),
	}
}

func (c *CacheMock) Put(name, key string, value []byte) {
	c.content[realKey(name, key)] = value
}

func (c *CacheMock) Get(name, key string) []byte {
	return c.content[realKey(name, key)]
}

func (c *CacheMock) Del(name, key string) []byte {
	delete(c.content, realKey(name, key))
	return []byte("Success")
}

func (c *CacheMock) GetByKey(key string) []byte {
	return c.content[key]
}

func (c *CacheMock) Keys() []string {
	sc := make([]string, 0)
	for k := range c.content {
		sc = append(sc, k)
	}
	return sc
}

func newLogger() protocol.Logger {
	cmLogger := logger.GetLogger("pubkey_manage")
	return cmLogger
}

func initTestEnv(t *testing.T) (*PubkeyManageRuntime, protocol.TxSimContext, func()) {
	pmRuntime := NewPubkeyManageRuntime(newLogger())
	ctrl := gomock.NewController(t)
	txSimContext := mock.NewMockTxSimContext(ctrl)

	cache := NewCacheMock()
	txSimContext.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			cache.Put(name, string(key), value)
			return nil
		}).AnyTimes()

	txSimContext.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return cache.Get(name, string(key)), nil
		}).AnyTimes()

	txSimContext.EXPECT().Del(gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return cache.Del(name, string(key)), nil
		}).AnyTimes()
	return pmRuntime, txSimContext, func() { ctrl.Finish() }
}

var strPubkey = "-----BEGIN PUBLIC KEY-----\n" +
	"MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAqM6qzNzBprMIqUbdDzkU\n" +
	"Fwz8rhi4/ASZB4UNMgxZlbeIZKDqOmQ8I6QjKQ5ZGGMzMeB6oEVv2TQ/8Az8F/mj\n" +
	"ok/w6vFrKM1m6j44W2x4DsO05jqgepOa9jr4Y4YSujOkMedS/mG3jSGLZtl+8nYB\n" +
	"UmquIoacLHNsFmmqB+CY2u1lQ0EaB3XQ/scoQzvtwN54OLUg5xOTViODb59a/w+P\n" +
	"ehu4YdTo5dLUY6idL24hCCVHIZQZMmBzg+lupxC4u8K5gPukEwcsYQ3IVhUHhvWC\n" +
	"AqtRxMywSo3aXfeo+7HJiWouK8dIsL+3VrTQdV8fn9/TImCFDORtyfCGlxLvS9J9\n" +
	"aUcpZrrk+qZUykZY3xRJ8pbhrgGghbmFJw/qCpeyTMfCooIdMezlBFoj2GdVLRLc\n" +
	"LEIief1a4Qg9sg9bJ6Dtj+tEMwjF5opLWm7x36zoakMAu9dQ/O6X0za+jPjY7hTO\n" +
	"BoACd/z8bWJEzVxb6jrFi+cGJY9i/n9CR4lWimzgYPQtAgMBAAE=\n" +
	"-----END PUBLIC KEY-----"

func TestAddPubkey(t *testing.T) {
	pmRuntime, context, fn := initTestEnv(t)
	defer fn()

	params := map[string][]byte{}

	params["pubkey"] = []byte(strPubkey)
	params["org_id"] = []byte("org1")
	params["role"] = []byte("client")
	result, err := pmRuntime.AddPubkey(context, params)
	if err != nil {
		t.Fatalf("AddPubkey error: %v", err)
	}

	fmt.Printf("result = %v \n", string(result))
}

func TestDelPubkey(t *testing.T) {
	pmRuntime, context, fn := initTestEnv(t)
	defer fn()

	params := map[string][]byte{}

	params["pubkey"] = []byte(strPubkey)
	params["org_id"] = []byte("org1")
	params["role"] = []byte("client")
	result, err := pmRuntime.AddPubkey(context, params)
	if err != nil {
		t.Fatalf("AddPubkey error: %v", err)
	}

	fmt.Printf("result = %v \n", string(result))

	delparams := map[string][]byte{}

	delparams["pubkey"] = []byte(strPubkey)

	result2, err := pmRuntime.DeletePubkey(context, delparams)

	if err != nil {
		t.Fatalf("DeletePubkey error: %v", err)
	}

	fmt.Printf("result = %v \n", string(result2))
}

func TestQueryPubkey(t *testing.T) {
	pmRuntime, context, fn := initTestEnv(t)
	defer fn()

	params := map[string][]byte{}

	params["pubkey"] = []byte(strPubkey)
	params["org_id"] = []byte("org1")
	params["role"] = []byte("client")
	result, err := pmRuntime.AddPubkey(context, params)
	if err != nil {
		t.Fatalf("AddPubkey error: %v", err)
	}

	fmt.Printf("result = %v \n", string(result))
	queryparams := map[string][]byte{}

	queryparams["pubkey"] = []byte(strPubkey)

	result2, err := pmRuntime.QueryPubkey(context, queryparams)
	if err != nil {
		t.Fatalf("QueryPubkey error: %v", err)
	}
	info := &accesscontrol.PKInfo{}
	if err = proto.Unmarshal(result2, info); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	fmt.Printf("org_id = %s, role = %s \n", info.OrgId, info.Role)
}

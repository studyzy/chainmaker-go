/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package pubkeymgr

import (
	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/mr-tron/base58"
	"strings"
)

const (
	//paramNameOrgId    = "org_id"
	paramNameRole     = "role"
	paramNamePubkey   = "pubkey"

)

type PubkeyManageContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewPubkeyManageContract(log protocol.Logger) *PubkeyManageContract {
	return &PubkeyManageContract{
		log:     log,
		methods: registerPubkeyManageContractMethods(log),
	}
}

func (c *PubkeyManageContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func registerPubkeyManageContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	// pubkey manager
	pubkeyManageRuntime := &PubkeyManageRuntime{log: log}

	methodMap[syscontract.PubkeyManageFunction_PUBKEY_ADD.String()] = pubkeyManageRuntime.AddPubkey
	methodMap[syscontract.PubkeyManageFunction_PUBKEY_DELETE.String()] = pubkeyManageRuntime.DeletePubkey
	methodMap[syscontract.PubkeyManageFunction_PUBKEY_QUERY.String()] = pubkeyManageRuntime.QueryPubkey

	return methodMap
}

type PubkeyManageRuntime struct {
	log protocol.Logger
}

func NewPubkeyManageRuntime(log protocol.Logger) *PubkeyManageRuntime {
	return &PubkeyManageRuntime{log: log}
}

func pubkeyHash(pubkey string) string {
	pkHash := sha256.Sum256([]byte(pubkey))
	strPkHash := base58.Encode(pkHash[:])
	return strPkHash
}

// Add pubkey
func (r *PubkeyManageRuntime) AddPubkey(context protocol.TxSimContext, params map[string][]byte) (
	result []byte, err error) {

	org_id := string(params[protocol.ConfigNameOrgId])
	if utils.IsAnyBlank(org_id) {
		err := fmt.Errorf("%s, param[org_id] of AddPubkey not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	role := string(params[paramNameRole])
	if utils.IsAnyBlank(role) {
		err := fmt.Errorf("%s, param[role] of AddPubkey not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	// check role
	upperRole := strings.ToUpper(role)
	if protocol.Role(upperRole) != protocol.RoleClient &&  protocol.Role(upperRole) != protocol.RoleLight &&  protocol.Role(upperRole) != protocol.RoleCommonNode {
		err := fmt.Errorf("%s, illegal param[role] of AddPubkey: %s", common.ErrParams.Error(), role)
		r.log.Errorf(err.Error())
		return nil, err
	}

	pubkey := string(params[paramNamePubkey])
	if utils.IsAnyBlank(pubkey) {
		err := fmt.Errorf("%s, param[pubkey] of AddPubkey not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	pkHashKey := pubkeyHash(pubkey)

	pkInfo := &accesscontrol.PKInfo{
		OrgId: org_id,
		Role: upperRole,
		PkPem: pubkey,
	}
//	value, err := pkInfo.Marshal()
	value, err := proto.Marshal(pkInfo)
	if err != nil {
		err := fmt.Errorf("marshal error in AddPubkey")
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(syscontract.SystemContract_PUBKEY_MANAGEMENT.String(), []byte(pkHashKey), value); err != nil {
		r.log.Errorf("Put failed in AddPubkey, err: %s", err.Error())
		return nil, err
	}

	r.log.Infof("pubkey add success")
	return []byte("Success"), nil
}

// Delete pubkey
func (r *PubkeyManageRuntime) DeletePubkey(context protocol.TxSimContext, params map[string][]byte) (
	result []byte, err error) {

	pubkey := string(params[paramNamePubkey])
	if utils.IsAnyBlank(pubkey) {
		err := fmt.Errorf("%s, param[pubkey] of DeletePubkey not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	pkHashKey := pubkeyHash(pubkey)
	bytes, err := context.Get(syscontract.SystemContract_PUBKEY_MANAGEMENT.String(), []byte(pkHashKey))
	if err != nil {
		r.log.Errorf("DeletePubkey get pubkey failed, pubkey[%s], err: %s", pubkey, err.Error())
		return nil, err
	}

	if len(bytes) == 0 {
		msg := fmt.Sprintf(			"DeletePubkey get pubkey failed, pubkey[%s], err: not exist", pubkey)
		r.log.Error(msg)
		return nil, errors.New(msg)
	}
	r.log.Infof("pubkey exists")

	err = context.Del(syscontract.SystemContract_PUBKEY_MANAGEMENT.String(), []byte(pkHashKey))
	if err != nil {
		r.log.Errorf("DeletePubkey for pubkey failed, pubkey[%s], err: %s", pubkey, err.Error())
		return nil, err
	}

	r.log.Infof("pubkey delete success")
	return []byte("Success"), nil
}

// Query pubkey
func (r *PubkeyManageRuntime) QueryPubkey(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {

	pubkey := string(params[paramNamePubkey])
	if utils.IsAnyBlank(pubkey) {
		err := fmt.Errorf("%s, param[pubkey] of QueryPubkey not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	pkHashKey := pubkeyHash(pubkey)
	bytes, err := context.Get(syscontract.SystemContract_PUBKEY_MANAGEMENT.String(), []byte(pkHashKey))
	if err != nil {
		r.log.Errorf("DeletePubkey get pubkey failed, pubkey[%s], err: %s", pubkey, err.Error())
		return nil, err
	}

	if len(bytes) == 0 {
		msg := fmt.Sprintf(			"DeletePubkey get pubkey failed, pubkey[%s], err: not exist", pubkey)
		r.log.Error(msg)
		return nil, errors.New(msg)
	}
	r.log.Infof("pubkey exists")

	return bytes, nil
}

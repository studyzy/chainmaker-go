/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/11/3
package native_test

import (
	"fmt"
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"

	"github.com/stretchr/testify/require"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	CHAIN1 = "chain1"
	isTls  = false
)

func getChainConfig() *configPb.ChainConfig {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config ============")
	// 构造Payload
	//pair := &commonPb.KeyValuePair{Key: "height", Value: strconv.FormatInt(1, 10)}
	var pairs []*commonPb.KeyValuePair
	//Pairs = append(Pairs, pair)

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.SystemContract_CHAIN_CONFIG.String(), MethodName: consts.ChainConfigManager_GET_CHAIN_CONFIG.String(), Pairs: pairs})
	if err == nil {
		result := &configPb.ChainConfig{}
		err = proto.Unmarshal(resp.ContractResult.Result, result)
		fmt.Printf("send tx resp: code:%d, msg:%s, chainConfig:%+v\n", resp.Code, resp.Message, result)
		return result
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println("WARN: client.call err: deadline")
		return nil
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
	return nil
}

// 查询链配置
func TestGetChainConfig(t *testing.T) {
	require.NotNil(t, getChainConfig())
}

// 根据blockHeight查询链配置
func TestGetChainConfigAt(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_height",
		Value: "0",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.SystemContract_CHAIN_CONFIG.String(), MethodName: consts.ChainConfigManager_GET_CHAIN_CONFIG_AT.String(), Pairs: pairs})
	if err == nil {
		fmt.Println(resp.ContractResult)
		result := &configPb.ChainConfig{}
		err = proto.Unmarshal(resp.ContractResult.Result, result)
		fmt.Printf("send tx resp: code:%d, msg:%s, chainConfig:%+v\n", resp.Code, resp.Message, result)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println("WARN: client.call err: deadline")
		return
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
}

var (
	orgId       = "org_id"
	templateStr = "\n============ send Tx [%s] ============\n"
)

// 更新Core配置
func TestUpdateCore(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_scheduler_timeout",
		Value: "15",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_scheduler_validate_timeout",
		Value: "20",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_CORE_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 更新Block配置
func TestUpdateBlock(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_timestamp_verify",
		Value: []byte("true"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_timeout",
		Value: "-1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_tx_capacity",
		Value: "10",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_size",
		Value: "10",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_interval",
		Value: "3000",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_BLOCK_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 根证书添加
func TestAddTrustRoot(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: orgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "root",
		Value: `
-----BEGIN CERTIFICATE-----
MIIDNjCCApigAwIBAgIDCAf8MAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTEwMzEyNDkzNloXDTMw
MTEwMTEyNDkzNlowgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzEuY2hhaW5t
YWtlci5vcmcwgZswEAYHKoZIzj0CAQYFK4EEACMDgYYABAAWyvxAG5reWbTWpd0Q
Bm5UDt4DVIuS8pjgEnvaVys/XTTB9DjvUQW28r6k22O2P4OLGq8lQ0puDEr7TiYr
JltzTQC/nEF/QtJjaRW98l32NqZzpjtVFTZy1jd7vqpIogbemq6zallwwXK0w792
zhuOMqb2q3ZXINRH4/I5nOTf/8zSGaOBpzCBpDAOBgNVHQ8BAf8EBAMCAaYwDwYD
VR0lBAgwBgYEVR0lADAPBgNVHRMBAf8EBTADAQH/MCkGA1UdDgQiBCAKogJqaxO0
df/ngy1+VfXPwM12k2Bk91uqyQbUFy50GTBFBgNVHREEPjA8gg5jaGFpbm1ha2Vy
Lm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnhwR/AAAB
MAoGCCqGSM49BAMCA4GLADCBhwJBee8wC03Wz6eV42KMMSHXa17tL/KNpVeCbLOo
oFhb8+WMRqqeAKNx61E5panzjqZR2ntvZ8AzvJpy7zUYtRZXeuQCQgHxil885sxo
+ha6Ty7ohEnKdK+JXO2hdI14QLsvEOTjA1Beul42U4XtNKCYZp+aNIjUUWIMAEKH
55yvulf9kDgsjw==
-----END CERTIFICATE-----
	`,
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_TRUST_ROOT_ADD.String(), pairs, chainConfig.Sequence)
}

// 根证书更新
func TestUpdateTrustRoot(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	// [注意]需要修改的组织需要和签名证书是一致的，否则无法修改
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "root",
		Value: `-----BEGIN CERTIFICATE-----
MIIDNjCCApigAwIBAgIDAeONMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnMi5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnMB4XDTIwMTEwMzEyNDkzN1oXDTMw
MTEwMTEyNDkzN1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcyLmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzIuY2hhaW5t
YWtlci5vcmcwgZswEAYHKoZIzj0CAQYFK4EEACMDgYYABADzIS7T4x788oWufjDb
u1AIStmGSvyzJpyq65mIcrxJddJAAZ3o+icnH+VhuEg6MJ1ErwYsD36b6yRozhxp
cHzJ7AFH0Fy9pBYH5S45M4nlOXEuKjFQj32KDufRhBLRq8k+MAsi+SEEOlE1cmWj
8lUXN23J9OqBBWi4FUuQovMUfR0hVaOBpzCBpDAOBgNVHQ8BAf8EBAMCAaYwDwYD
VR0lBAgwBgYEVR0lADAPBgNVHRMBAf8EBTADAQH/MCkGA1UdDgQiBCBzaApBM4pn
SgAEFDvUNydn0DbiWih7FUGLUqw7Yn18LjBFBgNVHREEPjA8gg5jaGFpbm1ha2Vy
Lm9yZ4IJbG9jYWxob3N0ghljYS53eC1vcmcyLmNoYWlubWFrZXIub3JnhwR/AAAB
MAoGCCqGSM49BAMCA4GLADCBhwJCAOn8dQoFtV0FuJhMKRsc2frkUdEHEeVIA6qe
VJVRsVYJOpWfn1/QWpYiWbn3eZMmQN6Y0LDEnzyuRuYZAYY8pBUZAkFFYsqJKqwC
Ac94P+IXMG3sBkeyq3wBzVxr8pCEaNVgVV0BUY240J/qW4vcBHqRyQ5ylppJ8RAo
uL8dAwVUqvB/iQ==
-----END CERTIFICATE-----
`,
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_TRUST_ROOT_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 根证书删除
func TestDeleteTrustRoot(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org2",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_TRUST_ROOT_DELETE.String(), pairs, chainConfig.Sequence)
}

// 节点地址添加
func TestAddNodeId(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: "QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ID_ADD.String(), pairs, chainConfig.Sequence)
}

// 节点ID更新
func TestUpdateNodeId(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: "QmecidwW22B2rPKe91smZFjKrbewwDgnHEbfBxydrzSMV2",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "new_node_id",
		Value: "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ID_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 节点地址删除
func TestDeleteNodeId(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: "QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ID_DELETE.String(), pairs, chainConfig.Sequence)
}

// 节点机构添加
func TestAddNodeOrg(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org3",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: "QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ORG_ADD.String(), pairs, chainConfig.Sequence)
}

// 节点机构更新
func TestUpdateNodeOrg(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org3",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: "QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ORG_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 节点机构删除
func TestDeleteNodeOrg(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org2",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_NODE_ORG_DELETE.String(), pairs, chainConfig.Sequence)
}

// 共识扩展字段的添加
func TestAddConsensusExt(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: "wx-org3",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_CONSENSUS_EXT_ADD.String(), pairs, chainConfig.Sequence)
}

// 共识扩展字段的更新
func TestUpdateConsensusExt(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: orgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "aa",
		Value: "chain01_ext",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_CONSENSUS_EXT_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 共识扩展字段的删除
func TestDeleteConsensusExt(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   orgId,
		Value: orgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "aa",
		Value: "chain01_ext",
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_CONSENSUS_EXT_DELETE.String(), pairs, chainConfig.Sequence)
}

// 权限添加
func TestPermissionAdd(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	p := &acPb.Policy{
		Rule:    string(protocol.RuleMajority),
		OrgList: []string{
			//"wx-org1",
		},
		RoleList: []string{
			//"Admin",
		},
	}
	pbStr, err := proto.Marshal(p)
	require.NoError(t, err)
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   consts.ChainConfigManager_NODE_ID_UPDATE.String(),
		Value: string(pbStr),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_PERMISSION_ADD.String(), pairs, chainConfig.Sequence)

}

// 权限修改
func TestPermissionUpdate(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	p := &acPb.Policy{
		Rule:    string(protocol.RuleMajority),
		OrgList: []string{
			//"wx-org1",
			//"wx-org2",
			//"wx-org3",
			//"wx-org4",
		},
		RoleList: []string{},
	}
	pbStr, err := proto.Marshal(p)
	require.NoError(t, err)

	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   consts.ChainConfigManager_NODE_ID_UPDATE.String(),
		Value: string(pbStr),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_PERMISSION_UPDATE.String(), pairs, chainConfig.Sequence)
}

// 权限删除
func TestPermissionDelete(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf(templateStr, txId)

	chainConfig := getChainConfig()
	require.NotNil(t, chainConfig, "chainConfig is empty")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: consts.ChainConfigManager_CORE_UPDATE.String(),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, commonPb.SystemContract_CHAIN_CONFIG.String(), consts.ChainConfigManager_PERMISSION_DELETE.String(), pairs, chainConfig.Sequence)
}

func processReq(txId string, txType commonPb.TxType, contractName, funcName string, pairs []*commonPb.KeyValuePair, sequence uint64) {
	sk, member := native.GetUserSK(1)
	resp, err := native.ConfigUpdateRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: txType, ChainId: CHAIN1,
		ContractName: contractName, MethodName: funcName, Pairs: pairs}, sequence)
	processResults(resp, err)
}

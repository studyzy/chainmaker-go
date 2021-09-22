/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"

	"github.com/stretchr/testify/require"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	CHAIN1 = "chain1"
	isTls  = true
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
		ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(), MethodName: syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), Pairs: pairs})
	if err == nil {
		if resp.Code != commonPb.TxStatusCode_SUCCESS {
			panic(resp.Message)
		}
		result := &configPb.ChainConfig{}
		err = proto.Unmarshal(resp.ContractResult.Result, result)
		data, _ := json.MarshalIndent(result, "", "\t")
		fmt.Printf("send tx resp: code:%d, msg:%s, chainConfig:%s\n", resp.Code, resp.Message, data)
		fmt.Printf("\n============ get chain config end============\n\n\n")
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
		Value: []byte("1"),
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(), MethodName: syscontract.ChainConfigFunction_GET_CHAIN_CONFIG_AT.String(), Pairs: pairs})
	if err == nil {
		result := &configPb.ChainConfig{}
		err = proto.Unmarshal(resp.ContractResult.Result, result)
		data, _ := json.MarshalIndent(result, "", "\t")
		fmt.Printf("send tx resp: code:%d, msg:%s, chainConfig:%s\n", resp.Code, resp.Message, data)
		fmt.Printf("\n============ get chain config end============\n\n\n")
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
		Value: []byte("16"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_scheduler_validate_timeout",
		Value: []byte("20"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_CORE_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("900"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_tx_capacity",
		Value: []byte("10"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_size",
		Value: []byte("10"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_interval",
		Value: []byte("3000"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org5.chainmaker.org"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "root",
		Value: []byte(`-----BEGIN CERTIFICATE-----
MIICrzCCAlWgAwIBAgIDCoJWMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTMw
MTIwNjA2NTM0M1owgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzUuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQvKJbsKIIfwZDBl7Fd
QFzub5HVLMYHbg9Vocg7FRiuOvggk9nR7kvRm8RD+AY64OpThhE5fCmYJLUhKr0Q
YyhFo4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEIEUAhxhcWZS15xG8t6OkdHY5bgbJhDdawNvk
X+ev1BPWMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh
Lnd4LW9yZzUuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIg
Joe9KHupPPSSQF7M+u0hmT/3TjHH1P9WkBItt0bFy1kCIQCCaRznhe1jnZ8kD8XS
7F36kC80dzJI7t6qhubcmUbt5A==
-----END CERTIFICATE-----
`),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org5.chainmaker.org"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "root",
		Value: []byte(`-----BEGIN CERTIFICATE-----
MIICrzCCAlWgAwIBAgIDAOetMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ
MA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt
b3JnNS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD
ExljYS53eC1vcmc1LmNoYWlubWFrZXIub3JnMB4XDTIxMDcyMjA5MzIzMVoXDTMx
MDcyMDA5MzIzMVowgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw
DgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmc1LmNoYWlubWFrZXIub3Jn
MRIwEAYDVQQLEwlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4LW9yZzUuY2hhaW5t
YWtlci5vcmcwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARezHsdPWKfxwtEx61G
Bce//KXsNJkdxuFamrIBDLe4XLuGaM791IqddFBaa6pDsPoaWPP+d2pDRuKekj1M
uoSRo4GnMIGkMA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEIMATPlrnbPNC94C3iK7EuhnBhQnZHaQI0/Vi
iTMzJYKiMEUGA1UdEQQ+MDyCDmNoYWlubWFrZXIub3Jngglsb2NhbGhvc3SCGWNh
Lnd4LW9yZzUuY2hhaW5tYWtlci5vcmeHBH8AAAEwCgYIKoZIzj0EAwIDSAAwRQIh
AOsYAbNJTT4GRVEOwpe6/yv3gomrb7bYmn0/o6myQcZQAiBxOtuRu3IihyK9PmEK
wrKB3vCIB2OTcU1bx3WKHi3W3Q==
-----END CERTIFICATE-----
`),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org2"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org1"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte("QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ID_ADD.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org1"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: []byte("QmecidwW22B2rPKe91smZFjKrbewwDgnHEbfBxydrzSMV2"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "new_node_id",
		Value: []byte("QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org1"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: []byte("QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org3"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte("QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org3"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte("QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org2"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte("wx-org3"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), pairs, chainConfig.Sequence)
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
		Value: []byte(orgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "aa",
		Value: []byte("chain01_ext"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Value: []byte(orgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "aa",
		Value: []byte("chain01_ext"),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), pairs, chainConfig.Sequence)
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
		Key:   syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(),
		Value: pbStr,
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_PERMISSION_ADD.String(), pairs, chainConfig.Sequence)

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
		Key:   syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(),
		Value: pbStr,
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), pairs, chainConfig.Sequence)
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
		Key: syscontract.ChainConfigFunction_CORE_UPDATE.String(),
	})
	processReq(txId, commonPb.TxType_INVOKE_CONTRACT, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), pairs, chainConfig.Sequence)
}

func processReq(txId string, txType commonPb.TxType, contractName, funcName string, pairs []*commonPb.KeyValuePair, sequence uint64) {
	sk, member := native.GetUserSK(1)
	resp, err := native.ConfigUpdateRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: txType, ChainId: CHAIN1,
		ContractName: contractName, MethodName: funcName, Pairs: pairs}, sequence)
	processResults(resp, err)
}

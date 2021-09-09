package native_test1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/test/common"
	"chainmaker.org/chainmaker/common/v2/crypto"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

var (
	orgId      = "org_id"
	timestamp  int64
	isRandTxId = true
)

func TestInstallContractMultiSignReq(t *testing.T) {
	isRandTxId = false
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_INIT_CONTRACT.String())
}
func TestInstallContractMultiSignVote(t *testing.T) {
	isRandTxId = false
	timestamp = 1630396944
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	testMultiSignVote(t, 2, txId, syscontract.ContractManageFunction_INIT_CONTRACT.String())
}
func TestContractInstallMultiSignQuery(t *testing.T) {
	isRandTxId = false
	timestamp = 1628771607
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_INIT_CONTRACT.String())
}

// test all native contract
func TestMultiSignAll(t *testing.T) {
	testContractInitMultiSign(t)
	testContractUpgradeMultiSign(t)
	testContractFreezeMultiSign(t)
	testContractUnfreezeMultiSign(t)
	testContractRevokeMultiSign(t)
	testCoreUpdateMultiSign(t)
	testBlockUpdateMultiSign(t)
	testTrustRootAddMultiSign(t)
	testTrustRootDeleteMultiSign(t)
	testNodeIdAddMultiSign(t)
	testNodeIdDeleteMultiSign(t)
	testNodeOrgAddMultiSign(t)
	testNodeOrgUpdateMultiSign(t)
	testNodeOrgDeleteMultiSign(t)
	testConsensusExtAddMultiSign(t)
	testConsensusExtUpdateMultiSign(t)
	testConsensusExtDeleteMultiSign(t)
	testPermissionAddMultiSign(t)
	testPermissionUpdateMultiSign(t)
	testPermissionDeleteMultiSign(t)
}

func testMultiSignReq(t *testing.T, method string) {
	common.SetCertPathPrefix(certPathPrefix)
	if isRandTxId {
		txId = utils.GetRandTxId()
	} else {
		txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	}
	log.Printf("timestamp:%d\n", timestamp)
	//builder.Write([]byte("b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877"))
	//builder.WriteByte(idFlag)
	//
	//sk, _ := native.GetUserSK(1)
	log.Printf("Req txId:%s\n", txId)
	var pairs1 []*commonPb.KeyValuePair
	switch method {
	case syscontract.ContractManageFunction_INIT_CONTRACT.String():
		pairs1 = initContractInitPairs() //构造交易发起pairs1
	case syscontract.ContractManageFunction_UPGRADE_CONTRACT.String():
		pairs1 = initContractUpgradePairs()
	case syscontract.ContractManageFunction_FREEZE_CONTRACT.String():
		pairs1 = initContractFreezePairs()
	case syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String():
		pairs1 = initContractUnfreezePairs()
	case syscontract.ContractManageFunction_REVOKE_CONTRACT.String():
		pairs1 = initContractRevokePairs()
	case syscontract.ChainConfigFunction_CORE_UPDATE.String():
		pairs1 = initCoreUpdatePairs()
	case syscontract.ChainConfigFunction_BLOCK_UPDATE.String():
		pairs1 = initBlockUpdatePairs()
	case syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String():
		pairs1 = initTrustRootAddPairs()
	case syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String():
		pairs1 = initTrustRootDeletePairs()
	case syscontract.ChainConfigFunction_NODE_ID_ADD.String():
		pairs1 = initNodeIdAddPairs()
	case syscontract.ChainConfigFunction_NODE_ID_DELETE.String():
		pairs1 = initNodeIdDeletePairs()
	case syscontract.ChainConfigFunction_NODE_ORG_ADD.String():
		pairs1 = initNodeOrgAddPairs()
	case syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String():
		pairs1 = initNodeOrgUpdatePairs()
	case syscontract.ChainConfigFunction_NODE_ORG_DELETE.String():
		pairs1 = initNodeOrgDeletePairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String():
		pairs1 = initConsensusExtAddPairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String():
		pairs1 = initConsensusExtUpdatePairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String():
		pairs1 = initConsensusExtDeletePairs()
	case syscontract.ChainConfigFunction_PERMISSION_ADD.String():
		pairs1 = initPermissionAddPairs()
	case syscontract.ChainConfigFunction_PERMISSION_UPDATE.String():
		pairs1 = initPermissionUpdatePairs()
	case syscontract.ChainConfigFunction_PERMISSION_DELETE.String():
		pairs1 = initPermissionDeletePairs()
		//case syscontract.CertManageFunction_CERTS_DELETE.String():
		// payload = initCertDeletePairs()
		//case syscontract.CertManageFunction_CERTS_UNFREEZE.String():
		// payload = initCertUnfreezePairs()
		//case syscontract.CertManageFunction_CERTS_REVOKE.String():
		// payload = initCertRevokePairs()
	}
	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   pairs1,
		TxId:         txId,
		ChainId:      CHAIN1,
	}
	//resp := common.ProposalMultiRequest(sk3, &client, payload.TxType,
	//CHAIN1, payload.TxId, payload, []int{1,2,3,4}, timestamp)
	resp := common.ProposalRequest(sk3, &client, commonPb.TxType_INVOKE_CONTRACT,
		CHAIN1, txId, payload, []int{1})
	fmt.Println("testMultiSignReq timestamp", timestamp)
	fmt.Println(resp)

}

func testMultiSignVote(t *testing.T, memberNum int, Id string, method string) {
	txId = Id
	log.Printf("timestamp:%d\n", timestamp)
	common.SetCertPathPrefix(certPathPrefix)
	log.Printf("memberNum:%d\n", memberNum)
	//sk, _ := native.GetUserSK(memberNum)
	var pairs1 []*commonPb.KeyValuePair
	//sk, _ := native.GetUserSK(1)
	switch method {
	case syscontract.ContractManageFunction_INIT_CONTRACT.String():
		pairs1 = initContractInitPairs() //构造交易发起pairs1
	case syscontract.ContractManageFunction_UPGRADE_CONTRACT.String():
		pairs1 = initContractUpgradePairs()
	case syscontract.ContractManageFunction_FREEZE_CONTRACT.String():
		pairs1 = initContractFreezePairs()
	case syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String():
		pairs1 = initContractUnfreezePairs()
	case syscontract.ContractManageFunction_REVOKE_CONTRACT.String():
		pairs1 = initContractRevokePairs()
	case syscontract.ChainConfigFunction_CORE_UPDATE.String():
		pairs1 = initCoreUpdatePairs()
	case syscontract.ChainConfigFunction_BLOCK_UPDATE.String():
		pairs1 = initBlockUpdatePairs()
	case syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String():
		pairs1 = initTrustRootAddPairs()
	case syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String():
		pairs1 = initTrustRootDeletePairs()
	case syscontract.ChainConfigFunction_NODE_ID_ADD.String():
		pairs1 = initNodeIdAddPairs()
	case syscontract.ChainConfigFunction_NODE_ID_DELETE.String():
		pairs1 = initNodeIdDeletePairs()
	case syscontract.ChainConfigFunction_NODE_ORG_ADD.String():
		pairs1 = initNodeOrgAddPairs()
	case syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String():
		pairs1 = initNodeOrgUpdatePairs()
	case syscontract.ChainConfigFunction_NODE_ORG_DELETE.String():
		pairs1 = initNodeOrgDeletePairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String():
		pairs1 = initConsensusExtAddPairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String():
		pairs1 = initConsensusExtUpdatePairs()
	case syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String():
		pairs1 = initConsensusExtDeletePairs()
	case syscontract.ChainConfigFunction_PERMISSION_ADD.String():
		pairs1 = initPermissionAddPairs()
	case syscontract.ChainConfigFunction_PERMISSION_UPDATE.String():
		pairs1 = initPermissionUpdatePairs()
	case syscontract.ChainConfigFunction_PERMISSION_DELETE.String():
		pairs1 = initPermissionDeletePairs()
	}
	payload1 := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   pairs1,
		TxId:         txId,
		ChainId:      CHAIN1,
	}
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	if err != nil {
		panic(err)
	}

	ee, err := AclSignOne(payloadBytes, memberNum) //单个用户对多签payload签名
	if err != nil {
		panic(err)
	}
	//构造多签投票信息
	msvi := &syscontract.MultiSignVoteInfo{
		Vote:        syscontract.VoteStatus_AGREE,
		Endorsement: ee,
	}
	msviByte, _ := msvi.Marshal()

	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiVote_VOTE_INFO.String(),
			Value: msviByte,
		},
		{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(payload1.TxId),
		},
	}

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_VOTE.String(),
		Parameters:   pairs,
		ChainId:      CHAIN1,
	}

	//resp := common.ProposalMultiRequest(sk3, &client, payload.TxType,
	// CHAIN1, "", payload, nil, time.Now().Unix())
	resp := common.ProposalRequest(sk3, &client, commonPb.TxType_INVOKE_CONTRACT,
		CHAIN1, "", payload, nil)

	fmt.Println(resp)
}

func testMultiSignQuery(t *testing.T, Id string, method string) {
	txId = Id
	common.SetCertPathPrefix(certPathPrefix)
	//sk, _ := native.GetAdminSK(1)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(txId),
		},
	}
	payload := &commonPb.Payload{
		TxId:         txId,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_QUERY.String(),
		Parameters:   pairs,
		ChainId:      CHAIN1,
	}

	resp := common.ProposalRequest(sk3, &client, payload.TxType,
		CHAIN1, "", payload, nil)

	fmt.Println(resp)
}

func testContractInitMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_INIT_CONTRACT.String())
	time.Sleep(4 * time.Second)
	//
	log.Printf("testContractInitMultiSign txId:%s\n", txId)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ContractManageFunction_INIT_CONTRACT.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_INIT_CONTRACT.String())
	var excepted = []byte("3")
	_, rst := testUpgradeInvokeSum(sk3, &client, CHAIN1)
	assert.Equal(t, excepted, rst)

}

func testContractUpgradeMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_UPGRADE_CONTRACT.String())
	time.Sleep(4 * time.Second)
	//
	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ContractManageFunction_UPGRADE_CONTRACT.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_UPGRADE_CONTRACT.String())

}

func testContractFreezeMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	time.Sleep(4 * time.Second)
	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())

}

func testContractUnfreezeMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	time.Sleep(4 * time.Second)
	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())

}

func testContractRevokeMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ContractManageFunction_REVOKE_CONTRACT.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ContractManageFunction_REVOKE_CONTRACT.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ContractManageFunction_REVOKE_CONTRACT.String())

}

func testUpgradeInvokeSum(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) (string, []byte) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sum][%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "arg1",
			Value: []byte("1"),
		},
		{
			Key:   "arg2",
			Value: []byte("2"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sum",
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload, nil)
	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId, resp.ContractResult.Result
}

func testCoreUpdateMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_CORE_UPDATE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_CORE_UPDATE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_CORE_UPDATE.String())
}

func testBlockUpdateMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_BLOCK_UPDATE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_BLOCK_UPDATE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_BLOCK_UPDATE.String())
}

func testTrustRootAddMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String())
}

func testTrustRootDeleteMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String())
}

func testNodeIdAddMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_NODE_ID_ADD.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_NODE_ID_ADD.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_NODE_ID_ADD.String())
}

func testNodeIdDeleteMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_NODE_ID_DELETE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_NODE_ID_DELETE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_NODE_ID_DELETE.String())
}

func testNodeOrgAddMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_NODE_ORG_ADD.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_NODE_ORG_ADD.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_NODE_ORG_ADD.String())
}

func testNodeOrgUpdateMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String())
}

func testNodeOrgDeleteMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_NODE_ORG_DELETE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_NODE_ORG_DELETE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_NODE_ORG_DELETE.String())
}

//共识扩展字段添加

func testConsensusExtAddMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String())
}

func testConsensusExtUpdateMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String())
}

func testConsensusExtDeleteMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String())
}

func testPermissionAddMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_PERMISSION_ADD.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_PERMISSION_ADD.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_PERMISSION_ADD.String())
}

func testPermissionUpdateMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_PERMISSION_UPDATE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_PERMISSION_UPDATE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_PERMISSION_UPDATE.String())
}

func testPermissionDeleteMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t, syscontract.ChainConfigFunction_PERMISSION_DELETE.String())
	time.Sleep(4 * time.Second)

	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId, syscontract.ChainConfigFunction_PERMISSION_DELETE.String())
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId, syscontract.ChainConfigFunction_PERMISSION_DELETE.String())
}

//func TestCertAdd(t *testing.T) {
// timestamp = time.Now().Unix()
// testMultiSignReq(t,syscontract.CertManageFunction_CERT_ADD.String())
// time.Sleep(4 * time.Second)
//
// for i := 2; i < 5; i++ {
//    testMultiSignVote(t, i, txId,syscontract.CertManageFunction_CERT_ADD.String())
// }
// time.Sleep(4 * time.Second)
// testMultiSignQuery(t, txId,syscontract.CertManageFunction_CERT_ADD.String())
//}

func initCoreUpdatePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_CORE_UPDATE.String()),
		},
		{
			Key:   "tx_scheduler_timeout",
			Value: []byte("16"),
		},
		{
			Key:   "tx_scheduler_validate_timeout",
			Value: []byte("20"),
		},
	}
	return pairs
}

func initBlockUpdatePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_BLOCK_UPDATE.String()),
		},
		{
			Key:   "tx_timestamp_verify",
			Value: []byte("true"),
		},
		{
			Key:   "tx_timeout",
			Value: []byte("900"),
		},
		{
			Key:   "block_tx_capacity",
			Value: []byte("10"),
		},
		{
			Key:   "block_size",
			Value: []byte("10"),
		},
		{
			Key:   "block_interval",
			Value: []byte("3000"),
		},
	}

	return pairs
}

func initTrustRootAddPairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org8.chainmaker.org"),
		},
		{
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
		},
	}
	return pairs
}

func initTrustRootDeletePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org8.chainmaker.org"),
		},
	}

	return pairs
}

func initNodeIdAddPairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_NODE_ID_ADD.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org1.chainmaker.org"),
		},
		{
			Key:   "node_ids",
			Value: []byte("QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
		},
	}

	return pairs
}

func initNodeIdDeletePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_NODE_ID_DELETE.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org1.chainmaker.org"),
		},
		{
			Key:   "node_id",
			Value: []byte("QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
		},
	}
	return pairs
}

func initNodeOrgAddPairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_NODE_ORG_ADD.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org3.chainmaker.org"),
		},
		{
			Key:   "node_ids",
			Value: []byte("QmdT1qXbJNovCSaXproaBCBAtecYshWHm2FELgd8A9M5WZ,QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
		},
	}

	return pairs
}

func initNodeOrgUpdatePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org3.chainmaker.org"),
		},
		{
			Key:   "node_ids",
			Value: []byte("QmPvhNFs29t1wyR989chECm8MrGj3w9f8qtuetoiLzxiyT"),
		},
	}

	return pairs
}

func initNodeOrgDeletePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_NODE_ORG_DELETE.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org3.chainmaker.org"),
		},
	}

	return pairs
}

func initConsensusExtAddPairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String()),
		},
		{
			Key:   orgId,
			Value: []byte("wx-org3.chainmaker.org"),
		},
	}
	return pairs
}

func initConsensusExtUpdatePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String()),
		},
		{
			Key:   orgId,
			Value: []byte(orgId),
		},
		{
			Key:   "aa",
			Value: []byte("chain01_ext"),
		},
	}
	return pairs
}

func initConsensusExtDeletePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String()),
		},
		{
			Key:   orgId,
			Value: []byte(orgId),
		},
		{
			Key:   "aa",
			Value: []byte("chain01_ext"),
		},
	}
	return pairs
}

func initPermissionAddPairs() []*commonPb.KeyValuePair {
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
	if err != nil {
		panic(err)
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_PERMISSION_ADD.String()),
		},
		{
			Key:   syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(),
			Value: pbStr,
		},
	}

	return pairs
}

func initPermissionUpdatePairs() []*commonPb.KeyValuePair {
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
	if err != nil {
		panic(err)
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_PERMISSION_UPDATE.String()),
		},
		{
			Key:   syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(),
			Value: pbStr,
		},
	}
	return pairs
}

func initPermissionDeletePairs() []*commonPb.KeyValuePair {

	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ChainConfigFunction_PERMISSION_DELETE.String()),
		},
		{
			Key: syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(),
		},
	}
	return pairs
}

func initContractInitPairs() []*commonPb.KeyValuePair {
	wasmBin, err := ioutil.ReadFile(WasmPath)
	if err != nil {
		panic(err)
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_INIT_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_VERSION.String(),
			Value: []byte("1.0"),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_BYTECODE.String(),
			Value: wasmBin,
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}

	return pairs
}

func initContractUpgradePairs() []*commonPb.KeyValuePair {
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_VERSION.String(),
			Value: []byte("2.0.1"),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_BYTECODE.String(),
			Value: wasmBin,
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}
	return pairs
}

func initContractFreezePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_FREEZE_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}
	return pairs
}

func initContractUnfreezePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}
	return pairs
}

func initContractRevokePairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_REVOKE_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}
	return pairs
}

func initCertAddPairs() []*commonPb.KeyValuePair {
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CERT_MANAGE.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.CertManageFunction_CERT_ADD.String()),
		},
	}
	return pairs
}

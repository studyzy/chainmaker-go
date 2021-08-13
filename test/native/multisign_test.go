package native

import (
	"chainmaker.org/chainmaker-go/test/common"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/crypto"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"crypto/md5"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

var (
	timestamp int64
)

func TestMultiSignReq(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t)
}
func testMultiSignReq(t *testing.T) {
	common.SetCertPathPrefix(certPathPrefix)
	//txId = utils.GetRandTxId()
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	log.Printf("txId:%s\n", txId)
	log.Printf("timestamp:%d\n", timestamp)
	payload := initPayload()

	resp := common.ProposalMultiRequest(sk3, &client, payload.TxType,
		CHAIN1, payload.TxId, payload, []int{1}, timestamp)
	fmt.Println("testMultiSignReq timestamp", timestamp)
	fmt.Println(resp)
}

func TestMultiSignVote(t *testing.T) {
	timestamp = 1628771607
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	testMultiSignVote(t, 2, txId)
}

func testMultiSignVote(t *testing.T, memberNum int, txId string) {
	log.Printf("timestamp:%d\n", timestamp)
	common.SetCertPathPrefix(certPathPrefix)
	log.Printf("memberNum:%d\n", memberNum)
	payload1 := initPayload()
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	fmt.Printf("testMultiSignVote1 payload md5 is %x \n", md5.Sum(payloadBytes))
	if err != nil {
		panic(err)
	}

	ee, err := AclSignOne(payloadBytes, memberNum)
	if err != nil {
		panic(err)
	}

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
	}

	resp := common.ProposalMultiRequest(sk3, &client, payload.TxType,
		CHAIN1, "", payload, nil, time.Now().Unix())

	fmt.Println(resp)
}
func TestMultiSignQuery(t *testing.T) {
	timestamp = 1628771607
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	testMultiSignQuery(t, txId)
}
func testMultiSignQuery(t *testing.T, txId string) {
	common.SetCertPathPrefix(certPathPrefix)
	payload1 := initPayload()
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	if err != nil {
		panic(err)
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "multiPayload",
			Value: payloadBytes,
		},
		{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(payload1.TxId),
		},
	}

	payload := &commonPb.Payload{
		TxId:         txId,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_QUERY.String(),
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, &client, payload.TxType,
		CHAIN1, "", payload, nil)

	fmt.Println(resp)
}

func TestMultiSign(t *testing.T) {
	timestamp = time.Now().Unix()
	testMultiSignReq(t)
	time.Sleep(4 * time.Second)
	txId = "b93bc2c1ac2d42398d8d90a414e1f3c03544defe9cb345578f970a7d51f7877a"
	for i := 2; i < 5; i++ {
		testMultiSignVote(t, i, txId)
	}
	time.Sleep(4 * time.Second)
	testMultiSignQuery(t, txId)
	var excepted []byte = []byte("3")
	_, rst := testUpgradeInvokeSum(sk3, &client, CHAIN1)
	assert.Equal(t, excepted, rst)

}

func initPayload() *commonPb.Payload {
	wasmBin, _ := ioutil.ReadFile(WasmPath)
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

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   pairs,
		TxId:         txId,
		ChainId:      CHAIN1,
	}
	return payload
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

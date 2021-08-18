package native_test

import (
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
)

var chainConfig *configPb.ChainConfig
var disabledContractListChainConfig []string

func init() {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		fmt.Errorf("failed to create Rpc connection with error: %v\n", err)
	}
	client = apiPb.NewRpcNodeClient(conn)
	chainConfig = getChainConfig()
	disabledContractListChainConfig = chainConfig.DisabledNativeContract
	fmt.Printf("disabled contract list from bc1.yml: %v\n", disabledContractListChainConfig)
}

func TestNativeContractAccessControl(t *testing.T) {
	var (
		txId  string
		block *commonPb.Block
		tx    *commonPb.Transaction
	)
	expectedDisabledContractList := disabledContractListChainConfig

	toAddContractList := []string{syscontract.SystemContract_DPOS_ERC20.String(), syscontract.SystemContract_DPOS_STAKE.String()}
	toRevokeContractList := []string{syscontract.SystemContract_CERT_MANAGE.String(),
		syscontract.SystemContract_GOVERNANCE.String(), syscontract.SystemContract_PRIVATE_COMPUTE.String()}

	testGetDisabledNativeContractList(t, expectedDisabledContractList)

	txId = testAddNativeContract(t, toAddContractList...)
	for block == nil {
		block = testGetBlockByTxId(t, client, txId)
	}

	tx = block.Txs[0]
	require.True(t, tx.Result.ContractResult.Code == 0)
	require.True(t, tx.Result.ContractResult.Message == "OK")
	block = nil

	expectedDisabledContractList = nil
	testGetDisabledNativeContractList(t, expectedDisabledContractList)

	txId = testRevokeNativeContract(t, toRevokeContractList...)
	for block == nil {
		block = testGetBlockByTxId(t, client, txId)
	}
	tx = block.Txs[0]
	require.True(t, tx.Result.ContractResult.Code == 0)
	require.True(t, tx.Result.ContractResult.Message == "OK")
	block = nil

	expectedDisabledContractList = append(expectedDisabledContractList, toRevokeContractList...)
	testGetDisabledNativeContractList(t, expectedDisabledContractList)

	txId = testVerifyContractAccessWithCertManage(t)
	for block == nil {
		block = testGetBlockByTxId(t, client, txId)
	}
	tx = block.Txs[0]
	require.True(t, tx.Result.ContractResult.Code == 1)
	require.True(t, tx.Result.ContractResult.Message == "Access Denied")
}

// Native合约list查询
func testGetDisabledNativeContractList(t *testing.T, expectedList []string) {
	fmt.Println("============ test get disabled contract list ===========")

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(), MethodName: syscontract.ContractQueryFunction_GET_DISABLED_CONTRACT_LIST.String(), Pairs: nil})
	processResults(resp, err)

	assert.Nil(t, err)
	disabledContractList := parseDisabledContractList(resp.ContractResult.Result)
	require.Equal(t, expectedList, disabledContractList)
	fmt.Printf("\n\n ========finished get disabled contract list======== \n ")
}

// 新增Native合约权限
func testAddNativeContract(t *testing.T, list ...string) string {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)

	fmt.Println("============test add native contract============")
	val, _ := json.Marshal(list)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "native_contract_name",
			Value: val,
		},
	}
	sk, member := native.GetUserSK(1)

	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(), MethodName: syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(), Pairs: pairs})
	processResults(resp, err)

	assert.Nil(t, err)
	fmt.Printf("\n\n ========end test add native contract======== \n ")
	return txId
}

// Revoke Native合约权限
func testRevokeNativeContract(t *testing.T, list ...string) string {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)

	fmt.Println("============test revoke native contract============")
	val, _ := json.Marshal(list)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "native_contract_name",
			Value: val,
		},
	}
	sk, member := native.GetUserSK(1)

	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(), MethodName: syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(), Pairs: pairs})
	processResults(resp, err)

	assert.Nil(t, err)
	fmt.Printf("\n\n ========end test revoke native contract======== \n ")
	return txId
}

// Native合约list查询
func testVerifyContractAccessWithCertManage(t *testing.T) string {
	fmt.Println("============ test verify contract access with Cert Manage ===========")

	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "cert_crl",
		// 多个就换行就行
		Value: []byte("-----BEGIN CRL-----\nMIIBXjCCAQMCAQEwCgYIKoZIzj0EAwIwgYoxCzAJBgNVBAYTAkNOMRAwDgYDVQQI\n" +
			"EwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNo\nYWlubWFrZXIub3JnMRIwEAYDVQQL" +
			"Ewlyb290LWNlcnQxIjAgBgNVBAMTGWNhLnd4\nLW9yZzEuY2hhaW5tYWtlci5vcmcXDTIxMDcyMDEyMjYzMloXDTIxMDcyMDE2MjYz\n" +
			"MlowFjAUAgMFL28XDTI0MDMyMzE1MDMwNVqgLzAtMCsGA1UdIwQkMCKAIDUkP3Ec\nubfENS6TH3DFczH5dAnC2eD73+wcUF" +
			"/bEIlnMAoGCCqGSM49BAMCA0kAMEYCIQDy\nwvxZL30HRdyQYJzb1HsczH9xnh3iY+aW1ZbY46KX8AIhAPw8140++BTkBnlKBtAH\n" +
			"PajXB4S3hsYlNv0RwV5Gfui4\n-----END CRL-----\n"),
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CERT_MANAGE.String(), MethodName: syscontract.CertManageFunction_CERTS_REVOKE.String(), Pairs: pairs})

	processResults(resp, err)
	fmt.Printf("\n\n ========finished test verify contract access with cert ======== \n ")
	return txId
}

func parseDisabledContractList(result []byte) []string {
	if string(result) == "null" {
		return nil
	}
	disabledContractList := string(result)
	disabledContractList = strings.Trim(strings.Trim(disabledContractList[1:len(disabledContractList)-1], "["), "]")
	disabledContractList = strings.Replace(disabledContractList, "\"", "", -1)
	r := strings.Split(disabledContractList, ",")
	return r
}

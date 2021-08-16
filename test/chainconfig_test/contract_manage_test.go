package native_test

import (
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
)

var client apiPb.RpcNodeClient
var chainConfig *configPb.ChainConfig

func init() {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		fmt.Errorf("failed to create Rpc connection with error: %v\n", err)
	}
	client = apiPb.NewRpcNodeClient(conn)
	chainConfig = getChainConfig()
	fmt.Printf("disabled contract list from bc1.yml: %v\n", chainConfig.DisabledNativeContract)
}

func TestGetDisabledNativeContractListTwice(t *testing.T) {
	fmt.Printf("\n						STEP (1/2)  ===>\n\n")
	TestGetDisabledNativeContractList(t)
	fmt.Println()
	fmt.Printf("\n						STEP (2/2) ===>\n\n")
	TestGetDisabledNativeContractList(t)

	fmt.Println()
	fmt.Println("						-------end---------")
}

// Native合约list查询
func TestGetDisabledNativeContractList(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ test get disabled contract list ===========")

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_CONTRACT, ChainId: CHAIN1,
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(), MethodName: syscontract.ContractQueryFunction_GET_DISABLED_CONTRACT_LIST.String(), Pairs: nil})
	processResults(resp, err)

	assert.Nil(t, err)
	//disabledContractList := string(resp.ContractResult.Result)
	fmt.Printf("\n\n ========finished get disabled contract list======== \n ")
	//fmt.Println(c)
	//assert.NotNil(t, c.CertInfos[0].Cert, "not found certs")
}

// 新增Native合约权限
func TestAddNativeContract(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)

	fmt.Println("============test add native contract============")
	val, _ := json.Marshal([]string{syscontract.SystemContract_DPOS_ERC20.String(), syscontract.SystemContract_DPOS_STAKE.String()})
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
}

// Revoke Native合约权限
func TestRevokeNativeContract(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)

	fmt.Println("============test add native contract============")
	val, _ := json.Marshal([]string{syscontract.SystemContract_CERT_MANAGE.String(), syscontract.SystemContract_GOVERNANCE.String(), syscontract.SystemContract_PRIVATE_COMPUTE.String()})
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
	fmt.Printf("\n\n ========end test add native contract======== \n ")
}

/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package contractmgr

import (
	"chainmaker.org/chainmaker-go/vm/native/chainconfigmgr"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"encoding/json"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker-go/vm/native/common"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	"chainmaker.org/chainmaker-go/utils"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

var (
	ContractName                   = syscontract.SystemContract_CONTRACT_MANAGE.String()
	keyContractName                = "_Native_Contract_List"
	contractsForMultiSignWhiteList = []string{
		syscontract.SystemContract_CERT_MANAGE.String(),
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		syscontract.SystemContract_CONTRACT_MANAGE.String(),
	}
)

type ContractManager struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewContractManager(log protocol.Logger) *ContractManager {
	return &ContractManager{
		log:     log,
		methods: registerContractManagerMethods(log),
	}
}

func (c *ContractManager) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func registerContractManagerMethods(log protocol.Logger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	runtime := &ContractManagerRuntime{log: log}
	methodMap[syscontract.ContractManageFunction_INIT_CONTRACT.String()] = runtime.installContract
	methodMap[syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()] = runtime.upgradeContract
	methodMap[syscontract.ContractManageFunction_FREEZE_CONTRACT.String()] = runtime.freezeContract
	methodMap[syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String()] = runtime.unfreezeContract
	methodMap[syscontract.ContractManageFunction_REVOKE_CONTRACT.String()] = runtime.revokeContract
	methodMap[syscontract.ContractQueryFunction_GET_CONTRACT_INFO.String()] = runtime.getContractInfo

	methodMap[syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String()] = runtime.grantContractAccess
	methodMap[syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String()] = runtime.revokeContractAccess
	methodMap[syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String()] = runtime.verifyContractAccess
	methodMap[syscontract.ContractQueryFunction_GET_DISABLED_CONTRACT_LIST.String()] = runtime.getDisabledContractList

	return methodMap

}

// enable access to a native contract
// this method will take off the contract name from the disabled contract list
func (r *ContractManagerRuntime) grantContractAccess(txSimContext protocol.TxSimContext,
	params map[string][]byte) ([]byte, error) {

	var (
		err                      error
		requestContractListBytes []byte
		updatedContractListBytes []byte
		disabledContractList     []string
		requestContractList      []string
		updatedContractList      []string
	)

	// 1. fetch the disabled contract list
	disabledContractList, err = r.fetchDisabledContractList(txSimContext)
	if err != nil {
		return nil, err
	}

	// 2. get the requested contracts to enable access from parameters
	requestContractListBytes = params["native_contract_name"]
	err = json.Unmarshal(requestContractListBytes, &requestContractList)
	if err != nil {
		return nil, err
	}

	// 3. adjust the disabled native contract list per the requested contract names
	updatedContractList = filterContracts(disabledContractList, requestContractList)
	updatedContractListBytes, err = json.Marshal(updatedContractList)
	if err != nil {
		return nil, err
	}

	// 4. store the adjusted native contract list back to the database
	err = storeDisabledContractList(txSimContext, updatedContractListBytes)
	if err != nil {
		return nil, err
	}

	r.log.Infof("grant access to contract: %v succeed!", requestContractList)
	return nil, nil
}

// disable access to a native contract
// this method will add the contract names to the disabled contract list
func (r *ContractManagerRuntime) revokeContractAccess(txSimContext protocol.TxSimContext,
	params map[string][]byte) ([]byte, error) {
	var (
		err                      error
		requestContractListBytes []byte
		updatedContractListBytes []byte
		disabledContractList     []string
		requestContractList      []string
		updatedContractList      []string
	)

	// 1. fetch the disabled contract list
	disabledContractList, err = r.fetchDisabledContractList(txSimContext)
	if err != nil {
		return nil, err
	}

	// 2. 2. get the requested contracts to disable access from parameters
	requestContractListBytes = params["native_contract_name"]
	err = json.Unmarshal(requestContractListBytes, &requestContractList)
	if err != nil {
		return nil, err
	}

	// 3. adjust the disabled native contract list per the requested contract names
	updatedContractList = append(disabledContractList, requestContractList...)

	updatedContractListBytes, err = json.Marshal(updatedContractList)
	if err != nil {
		return nil, err
	}

	// 4. store the updated native contract list back to the database
	err = storeDisabledContractList(txSimContext, updatedContractListBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// verify if access to the requested contract is enabled according to the disabled contract list
// returns true as []byte if so or false otherwise
func (r *ContractManagerRuntime) verifyContractAccess(txSimContext protocol.TxSimContext,
	params map[string][]byte) ([]byte, error) {
	var (
		err                   error
		disabledContractList  []string
		contractName          string
		multiSignContractName string
	)

	// 1. fetch the disabled native contract list
	disabledContractList, err = r.fetchDisabledContractList(txSimContext)
	if err != nil {
		return nil, err
	}

	// 2. get the requested contract name and verify if it's on the disabled native contract list
	contractName = txSimContext.GetTx().Payload.ContractName
	for _, cn := range disabledContractList {
		if cn == contractName {
			return []byte("false"), nil
		}
	}

	// 3. if the requested contract name is multisignature, get the underlying contract name from
	// the tx payload and verify if it has access
	if contractName == syscontract.SystemContract_MULTI_SIGN.String() {

		// if the method name is not req, return true since it does not contain contract names in parameters
		multiSignMethodName := txSimContext.GetTx().Payload.Method
		if multiSignContractName == syscontract.MultiSignFunction_QUERY.String() ||
			multiSignMethodName == syscontract.MultiSignFunction_VOTE.String() {
			return []byte("true"), nil
		}

		multiSignContractName, err = getContractNameForMultiSign(txSimContext.GetTx().Payload.Parameters)
		if err != nil {
			return nil, err
		}

		// check if the requested contract is on the disabled contract list
		for _, cn := range disabledContractList {
			if cn == multiSignContractName {
				return []byte("false"), nil
			}
		}

		// check if the requested contract list is on the multi sign white list
		for _, cn := range contractsForMultiSignWhiteList {
			if cn == multiSignContractName {
				return []byte("true"), nil
			}
		}

		return []byte("false"), nil
	}

	return []byte("true"), nil
}

func getContractNameForMultiSign(params []*commonPb.KeyValuePair) (string, error) {
	for i, pair := range params {
		if pair.Key == syscontract.MultiReq_SYS_CONTRACT_NAME.String() {
			return string(params[i].Value), nil
		}
	}
	return "", errors.New("can't find the contract name for multi sign")
}

// fetch the disabled contract list
func (r *ContractManagerRuntime) getDisabledContractList(txSimContext protocol.TxSimContext,
	params map[string][]byte) ([]byte, error) {
	var (
		err                       error
		disabledContractList      []string
		disabledContractListBytes []byte
	)

	disabledContractList, err = r.fetchDisabledContractList(txSimContext)
	fmt.Printf("the result is %v\n", disabledContractList)

	if err != nil {
		return nil, err
	}

	disabledContractListBytes, err = json.Marshal(disabledContractList)

	if err != nil {
		return nil, err
	}
	return disabledContractListBytes, nil
}

// store the disabled contract list to the database
func storeDisabledContractList(txSimContext protocol.TxSimContext, disabledContractListBytes []byte) error {
	var (
		err                              error
		refinedDisabledContractListBytes []byte
		disabledContractList             []string
		refinedDisabledContractList      []string
	)
	err = json.Unmarshal(disabledContractListBytes, &disabledContractList)
	if err != nil {
		return err
	}

	// filter out redundant contract names in the disabled contract list
	uniqueMap := make(map[string]string)
	for _, cn := range disabledContractList {
		if _, ok := uniqueMap[cn]; !ok {
			uniqueMap[cn] = cn
			refinedDisabledContractList = append(refinedDisabledContractList, cn)
		}
	}

	refinedDisabledContractListBytes, err = json.Marshal(refinedDisabledContractList)
	if err != nil {
		return err
	}

	err = txSimContext.Put(ContractName, []byte(keyContractName), refinedDisabledContractListBytes)
	if err != nil {
		return err
	}
	return nil
}

// filter out request contracts from disabled contract list, return a newly updated list
func filterContracts(disabledContractList []string, requestedContractList []string) []string {
	var updatedContractList []string

	// return the original list if no contracts have been requested
	if len(requestedContractList) == 0 {
		return disabledContractList
	}

	m := make(map[string]int, len(requestedContractList))
	for _, cn := range requestedContractList {
		m[cn] = 1
	}

	// populate the updatedContractList
	for _, cn := range disabledContractList {
		_, found := m[cn]
		if !found {
			updatedContractList = append(updatedContractList, cn)
		}
	}

	return updatedContractList
}

// helper method to fetch the disabled contract list from genesis config file
// if not initialized or from the database otherwise
func (r *ContractManagerRuntime) fetchDisabledContractList(txSimContext protocol.TxSimContext) ([]string, error) {
	// try to get disabled contract list from database
	disabledContractListBytes, err := txSimContext.Get(ContractName, []byte(keyContractName))
	if err != nil {
		return nil, err
	}

	// if the config file does not exist in the database yet, try fetch it from the genesis config file and store it
	// to the database
	if disabledContractListBytes == nil {
		disabledContractListBytes, err = r.initializeDisabledNativeContractList(txSimContext)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
	}

	var disabledContractList []string
	err = json.Unmarshal(disabledContractListBytes, &disabledContractList)
	return disabledContractList, err
}

func (r *ContractManagerRuntime) getContractInfo(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name := string(parameters[syscontract.GetContractInfo_CONTRACT_NAME.String()])
	contract, err := r.GetContractInfo(txSimContext, name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(contract)
}

func (r *ContractManagerRuntime) initializeDisabledNativeContractList(
	txSimContext protocol.TxSimContext) ([]byte, error) {
	var (
		err                       error
		chainConfig               *configPb.ChainConfig
		disabledContractListBytes []byte
	)

	// 1. fetch chainConfig from genesis config file
	chainConfig, err = chainconfigmgr.GetChainConfig(txSimContext, make(map[string][]byte))
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	disabledContractList := chainConfig.DisabledNativeContract
	disabledContractListBytes, err = json.Marshal(disabledContractList)
	if err != nil {
		return nil, err
	}

	// 2. store the disabledContractList field to the database
	err = storeDisabledContractList(txSimContext, disabledContractListBytes)
	if err != nil {
		return nil, err
	}

	return disabledContractListBytes, nil
}

//func (r *ContractManagerRuntime) getAllContracts(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
//	[]byte, error) {
//	contracts, err := r.GetAllContracts(txSimContext)
//	if err != nil {
//		return nil, err
//	}
//	return json.Marshal(contracts)
//}
func (r *ContractManagerRuntime) installContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name, version, byteCode, runtimeType, err := r.parseParam(parameters)
	if err != nil {
		return nil, err
	}
	contract, err := r.InstallContract(txSimContext, name, version, byteCode, runtimeType, parameters)
	if err != nil {
		return nil, err
	}
	r.log.Infof("install contract success[name:%s version:%s runtimeType:%d byteCodeLen:%d]", contract.Name,
		contract.Version, contract.RuntimeType, len(byteCode))
	return contract.Marshal()
}

func (r *ContractManagerRuntime) upgradeContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name, version, byteCode, runtimeType, err := r.parseParam(parameters)
	if err != nil {
		return nil, err
	}
	contract, err := r.UpgradeContract(txSimContext, name, version, byteCode, runtimeType, parameters)
	if err != nil {
		return nil, err
	}
	r.log.Infof("upgrade contract success[name:%s version:%s runtimeType:%d byteCodeLen:%d]", contract.Name,
		contract.Version, contract.RuntimeType, len(byteCode))
	return contract.Marshal()
}

func (r *ContractManagerRuntime) parseParam(parameters map[string][]byte) (string, string, []byte,
	commonPb.RuntimeType, error) {
	name := string(parameters[syscontract.InitContract_CONTRACT_NAME.String()])
	version := string(parameters[syscontract.InitContract_CONTRACT_VERSION.String()])
	byteCode := parameters[syscontract.InitContract_CONTRACT_BYTECODE.String()]
	runtime := parameters[syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String()]
	if utils.IsAnyBlank(name, version, byteCode, runtime) {
		return "", "", nil, 0, errors.New("params contractName/version/byteCode/runtimeType cannot be empty")
	}
	runtimeInt := commonPb.RuntimeType_value[string(runtime)]
	if runtimeInt == 0 || int(runtimeInt) >= len(commonPb.RuntimeType_value) {
		return "", "", nil, 0, errors.New("params runtimeType[" + string(runtime) + "] is error")
	}
	runtimeType := commonPb.RuntimeType(runtimeInt)
	return name, version, byteCode, runtimeType, nil
}

func (r *ContractManagerRuntime) freezeContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name := string(parameters[syscontract.GetContractInfo_CONTRACT_NAME.String()])
	contract, err := r.FreezeContract(txSimContext, name)
	if err != nil {
		return nil, err
	}
	r.log.Infof("freeze contract success[name:%s version:%s runtimeType:%d]", contract.Name, contract.Version,
		contract.RuntimeType)
	return json.Marshal(contract)
}
func (r *ContractManagerRuntime) unfreezeContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name := string(parameters[syscontract.GetContractInfo_CONTRACT_NAME.String()])
	contract, err := r.UnfreezeContract(txSimContext, name)
	if err != nil {
		return nil, err
	}
	r.log.Infof("unfreeze contract success[name:%s version:%s runtimeType:%d]", contract.Name, contract.Version,
		contract.RuntimeType)
	return json.Marshal(contract)
}
func (r *ContractManagerRuntime) revokeContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (
	[]byte, error) {
	name := string(parameters[syscontract.GetContractInfo_CONTRACT_NAME.String()])
	contract, err := r.RevokeContract(txSimContext, name)
	if err != nil {
		return nil, err
	}
	r.log.Infof("revoke contract success[name:%s version:%s runtimeType:%d]", contract.Name, contract.Version,
		contract.RuntimeType)
	return json.Marshal(contract)
}

type ContractManagerRuntime struct {
	log protocol.Logger
}

//GetContractInfo 根据合约名字查询合约的详细信息
func (r *ContractManagerRuntime) GetContractInfo(context protocol.TxSimContext, name string) (*commonPb.Contract,
	error) {
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] of get contract not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	return utils.GetContractByName(context.Get, name)
}
func (r *ContractManagerRuntime) GetContractByteCode(context protocol.TxSimContext, name string) ([]byte, error) {
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] of get contract not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	return utils.GetContractBytecode(context.Get, name)
}

//GetAllContracts 查询所有合约的详细信息
func (r *ContractManagerRuntime) GetAllContracts(context protocol.TxSimContext) ([]*commonPb.Contract, error) {
	keyPrefix := []byte(utils.PrefixContractInfo)
	it, err := context.Select(syscontract.SystemContract_CONTRACT_MANAGE.String(), keyPrefix, keyPrefix)
	if err != nil {
		return nil, err
	}
	defer it.Release()
	var result []*commonPb.Contract
	for it.Next() {
		contract := &commonPb.Contract{}
		kv, err := it.Value()
		if err != nil {
			return nil, err
		}
		err = contract.Unmarshal(kv.Value)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

//InstallContract 安装新合约
func (r *ContractManagerRuntime) InstallContract(context protocol.TxSimContext, name, version string, byteCode []byte,
	runTime commonPb.RuntimeType, initParameters map[string][]byte) (*commonPb.Contract, error) {
	if !utils.CheckContractNameFormat(name) {
		return nil, errInvalidContractName
	}
	if runTime == commonPb.RuntimeType_EVM && !utils.CheckEvmAddressFormat(name) {
		return nil, errInvalidEvmContractName
	}
	key := utils.GetContractDbKey(name)
	//check name exist
	existContract, _ := context.Get(ContractName, key)
	if len(existContract) > 0 { //exist
		return nil, errContractExist
	}
	contract := &commonPb.Contract{
		Name:        name,
		Version:     version,
		RuntimeType: runTime,
		Status:      commonPb.ContractStatus_NORMAL,
		Creator:     context.GetSender(),
	}
	cdata, _ := contract.Marshal()

	err := context.Put(ContractName, key, cdata)
	if err != nil {
		return nil, err
	}
	byteCodeKey := utils.GetContractByteCodeDbKey(name)
	err = context.Put(ContractName, byteCodeKey, byteCode)
	if err != nil {
		return nil, err
	}
	//实例化合约，并init合约，产生读写集
	result, statusCode := context.CallContract(contract, protocol.ContractInitMethod, byteCode, initParameters,
		0, commonPb.TxType_INVOKE_CONTRACT)
	if statusCode != commonPb.TxStatusCode_SUCCESS {
		return nil, errContractInitFail
	}
	if result.Code > 0 { //throw error
		return nil, errContractInitFail
	}
	if runTime == commonPb.RuntimeType_EVM {
		//save bytecode body
		//EVM的特殊处理，在调用构造函数后会返回真正需要存的字节码，这里将之前的字节码覆盖
		if len(result.Result) > 0 {
			err := context.Put(ContractName, byteCodeKey, result.Result)
			if err != nil {
				return nil, errContractInitFail
			}
		}
	}
	return contract, nil
}

//UpgradeContract 升级现有合约
func (r *ContractManagerRuntime) UpgradeContract(context protocol.TxSimContext, name, version string, byteCode []byte,
	runTime commonPb.RuntimeType, upgradeParameters map[string][]byte) (*commonPb.Contract, error) {
	key := utils.GetContractDbKey(name)
	//check name exist
	existContract, _ := context.Get(ContractName, key)
	if len(existContract) == 0 { //not exist
		return nil, errContractNotExist
	}
	contract := &commonPb.Contract{}
	err := contract.Unmarshal(existContract)
	if err != nil {
		return nil, err
	}
	if contract.Version == version {
		return nil, errContractVersionExist
	}
	contract.RuntimeType = runTime
	contract.Version = version
	//update ContractInfo
	cdata, _ := contract.Marshal()
	err = context.Put(ContractName, key, cdata)
	if err != nil {
		return nil, err
	}
	//update Contract Bytecode
	byteCodeKey := utils.GetContractByteCodeDbKey(name)
	err = context.Put(ContractName, byteCodeKey, byteCode)
	if err != nil {
		return nil, err
	}
	//运行新合约的upgrade方法，产生读写集
	result, statusCode := context.CallContract(contract, protocol.ContractUpgradeMethod, byteCode, upgradeParameters,
		0, commonPb.TxType_INVOKE_CONTRACT)
	if statusCode != commonPb.TxStatusCode_SUCCESS {
		return nil, errContractUpgradeFail
	}
	if result.Code > 0 { //throw error
		return nil, errContractUpgradeFail
	}
	if runTime == commonPb.RuntimeType_EVM {
		//save bytecode body
		//EVM的特殊处理，在调用构造函数后会返回真正需要存的字节码，这里将之前的字节码覆盖
		if len(result.Result) > 0 {
			err := context.Put(ContractName, byteCodeKey, result.Result)
			if err != nil {
				return nil, errContractUpgradeFail
			}
		}
	}
	return contract, nil
}
func (r *ContractManagerRuntime) FreezeContract(context protocol.TxSimContext, name string) (
	*commonPb.Contract, error) {
	return r.changeContractStatus(context, name, commonPb.ContractStatus_NORMAL, commonPb.ContractStatus_FROZEN)
}
func (r *ContractManagerRuntime) UnfreezeContract(context protocol.TxSimContext, name string) (
	*commonPb.Contract, error) {
	return r.changeContractStatus(context, name, commonPb.ContractStatus_FROZEN, commonPb.ContractStatus_NORMAL)
}
func (r *ContractManagerRuntime) RevokeContract(context protocol.TxSimContext, name string) (
	*commonPb.Contract, error) {
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	contract, err := utils.GetContractByName(context.Get, name)
	if err != nil {
		return nil, err
	}
	if contract.Status != commonPb.ContractStatus_NORMAL && contract.Status != commonPb.ContractStatus_FROZEN {
		r.log.Errorf("contract[%s] expect status:NORMAL or FROZEN,actual status:%s",
			name, contract.Status.String())
		return nil, errContractStatusInvalid
	}
	contract.Status = commonPb.ContractStatus_REVOKED
	key := utils.GetContractDbKey(name)
	cdata, _ := contract.Marshal()
	err = context.Put(ContractName, key, cdata)
	if err != nil {
		return nil, err
	}
	return contract, nil
}

func (r *ContractManagerRuntime) changeContractStatus(context protocol.TxSimContext, name string,
	oldStatus, newStatus commonPb.ContractStatus) (*commonPb.Contract, error) {
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	contract, err := utils.GetContractByName(context.Get, name)
	if err != nil {
		return nil, err
	}
	if contract.Status != oldStatus {
		r.log.Errorf("contract[%s] expect status:%s,actual status:%s",
			name, oldStatus.String(), contract.Status.String())
		return nil, errContractStatusInvalid
	}
	contract.Status = newStatus
	key := utils.GetContractDbKey(name)
	cdata, _ := contract.Marshal()
	err = context.Put(ContractName, key, cdata)
	if err != nil {
		return nil, err
	}
	return contract, nil
}

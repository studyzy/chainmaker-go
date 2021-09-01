/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	evm "chainmaker.org/chainmaker/common/v2/evmutils"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

//func getByteCode(bytecode []byte, runtime commonPb.RuntimeType) ([]*commonPb.KeyValuePair, error) {
//	if runtime != commonPb.RuntimeType_EVM  {
//		return pairs, nil
//	}
//}

var printEnable = false

type EvmAbi struct {
	anonymous       bool         `json:"anonymous"`
	constant        bool         `json:"constant"`
	inputs          []*inOutType `json:"inputs"`
	name            string       `json:"name"`
	outputs         []*inOutType `json:"outputs"`
	payable         bool         `json:"payable"`
	stateMutability string       `json:"stateMutability"`
	typeEvm         string       `json:"typeEvm"`
}

type inOutType struct {
	indexed bool   `json:"indexed"`
	name    string `json:"name"`
	typeEvm string `json:"typeEvm"`
}

//
//func getValue(method string, abiPath string, respData []byte, runtime commonPb.RuntimeType) ([]byte, error) {
//
//	if runtime != commonPb.RuntimeType_EVM  {
//		return respData, nil
//	}
//
//	var params []interface{}
//	abiJson, err := ioutil.ReadFile(abiPath)
//	if err != nil {
//		return respData, nil
//	}
//	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
//	if err != nil {
//		return respData, nil
//	}
//	var evmAbi []map[string]interface{}
//	//abiJson2 := strings.ReplaceAll(abiJson, "\"type\"", "\"typeEvm\"")
//	err = json.Unmarshal([]byte(abiJson), &evmAbi)
//	if err != nil {
//		return nil, err
//	}
//
//	var methodObj []map[string]interface{}
//	for _, obj := range evmAbi {
//		if obj["name"].(string) == method {
//			tmp := obj["inputs"].([]interface{})
//			for _, t := range tmp {
//				methodObj = append(methodObj, t.(map[string]interface{}))
//			}
//			break
//		}
//	}
//
//	dataByte, _ := myAbi.Unpack(method, respData)
//
//	for _, pair := range dataByte {
//		for _, paramMap := range methodObj {
//			if pair.Key == paramMap["name"].(string) {
//				switch paramMap["type"].(string) {
//				case "address":
//					ski := pair.Value
//					if strings.LastIndex(pair.Value, ".crt") > -1 {
//						ski, err = getSki(pair.Value)
//						if err != nil {
//							return nil, err
//						}
//					}
//					add, err := getAddr(ski)
//					if err != nil {
//						return nil, err
//					}
//					params = append(params, add)
//				case "uint256", "int256", "uint", "int", "uint8", "int8":
//					val, err := strconv.Atoi(pair.Value)
//					if err != nil {
//						return nil, err
//					}
//					params = append(params, big.NewInt(int64(val)))
//				case "bool":
//					if strings.ToUpper(pair.Value) == "TRUE" {
//						params = append(params, true)
//					} else {
//						params = append(params, false)
//					}
//				case "bytes32", "byte", "bytes1":
//				case "bytes", "string":
//
//				}
//			}
//		}
//	}
//}
// Strval 获取变量的字符串值
// 浮点型 3.0将会转换成字符串3, "3"
// 非数值或字符类型的变量将会被转换成JSON格式字符串
func getStrval(value interface{}) string {
	// interface 转 string
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}

func makePairs(method string, abiPath string, pairs []*commonPb.KeyValuePair, runtime commonPb.RuntimeType, abiData *[]byte) (string, []*commonPb.KeyValuePair, error) {
	if runtime != commonPb.RuntimeType_EVM {
		return method, pairs, nil
	}
	data := ""
	if printEnable {
		fmt.Println("pairs: ", pairs, ", method: ", method)
		fmt.Println("[start]abiPath: ", abiPath, ", abiData: ", abiData)
	}

	var abiJson []byte
	var err error
	var params []interface{}
	if abiData == nil {
		abiJson, err = ioutil.ReadFile(abiPath)
		if printEnable {
			fmt.Println("[readfile]abiData: ", abiData, ", abiJson: ", abiJson)
		}
		if err != nil {
			return method, nil, err
		}
	} else {
		abiJson = *abiData
	}

	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		return method, nil, err
	}

	if printEnable {
		fmt.Println("myAbi: ", myAbi, ", abiJson: ", abiJson)
	}

	if len(pairs) > 0 {
		var evmAbi []map[string]interface{}
		//abiJson2 := strings.ReplaceAll(abiJson, "\"type\"", "\"typeEvm\"")
		err = json.Unmarshal([]byte(abiJson), &evmAbi)
		if err != nil {
			return method, nil, err
		}

		var methodObj []map[string]interface{}
		for _, obj := range evmAbi {

			if printEnable {
				fmt.Println("obj: ", obj)
			}

			objName, ok := obj["name"]
			// 匿名函数
			if !ok {
				continue
			}

			if objName.(string) == method {
				tmp := obj["inputs"].([]interface{})
				for _, t := range tmp {
					methodObj = append(methodObj, t.(map[string]interface{}))
				}
				break
			}
		}
		if printEnable {
			fmt.Println("methodObj: ", methodObj)
		}
		for _, paramMap := range methodObj {

			for _, pair := range pairs {

				if printEnable {
					fmt.Println("pair: ", pair, ", paramMap:", paramMap, ",paramMap[\"type\"].(string): ", paramMap["type"].(string), ", paramMap[\"name\"].(string):", paramMap["name"].(string), ", type(pair):", reflect.TypeOf(pair.Value), ", params: ", params)
				}

				if paramMap["name"].(string) == pair.Key {
					switch paramMap["type"].(string) {
					case "address":
						var add *evm.Address
						ski := string(pair.Value)
						if printEnable {
							fmt.Println("[debug] pair.Key: ", pair.Key, "pair.Value:", pair.Value, ", ski:", ski, ", typeof(ski): ", reflect.TypeOf(ski))
						}
						if strings.LastIndex(string(pair.Value), ".crt") > -1 {
							ski, err = getSki(string(pair.Value))
							if err != nil {
								return method, nil, err
							}
							add, err = getAddr(ski)
							if err != nil {
								return method, nil, err
							}
						} else {
							add, err = getContractAddress(ski)
							if printEnable {
								fmt.Println("[debug] pair.Key: ", pair.Key, "ski:", ski, ", add: ", add)
							}
							if err != nil {
								return method, nil, err
							}
						}
						params = append(params, add)
						if printEnable {
							fmt.Println("[append][address] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, "type(add):", reflect.TypeOf(add), ", add: ", add, ", params:", params)
						}
					case "uint256", "int256", "uint", "int", "uint8", "int8":
						val, err := strconv.Atoi(string(pair.Value))
						if err != nil {
							return method, nil, err
						}
						params = append(params, big.NewInt(int64(val)))
						if printEnable {
							fmt.Println("[append][uint] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", val:", val, "type(val):", reflect.TypeOf(val), ",big.NewInt(int64(val)): ", big.NewInt(int64(val)), ", params:", params)
						}
					//case "uint", "int", "uint8", "int8":
					//	val, err := strconv.Atoi(pair.Value)
					//	if err != nil {
					//		return method, nil, err
					//	}
					//	params = append(params, big.NewInt(int64(val)))
					case "bool":
						if strings.ToUpper(string(pair.Value)) == "TRUE" {
							params = append(params, true)
						} else {
							params = append(params, false)
						}
						if printEnable {
							fmt.Println("[append][bool] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					case "bytes32", "byte", "bytes1":
						params = append(params, string(pair.Value))
						if printEnable {
							fmt.Println("[append][bytes] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					case "bytes", "string":
						params = append(params, string(pair.Value))
						if printEnable {
							fmt.Println("[append][string] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					}
				}
			}
		}
	}
	if printEnable {
		fmt.Println("change: Pack para: ", method, params)
	}
	dataByte, err := myAbi.Pack(method, params...)
	if printEnable {
		fmt.Println("change: Pack para: ", method, params, " ---> dataByte: ", dataByte, ", err: ", err)
	}
	if err != nil {
		return method, nil, err
	}
	data = hex.EncodeToString(dataByte)
	//fmt.Println("Pack data: ", data)
	if len(data) != 0 {
		method = data[0:8]
	}
	var result []*commonPb.KeyValuePair
	result = []*commonPb.KeyValuePair{
		{
			Key:   "data",
			Value: []byte(data),
		},
	}
	if printEnable {
		fmt.Println("Pack para: ", method, params, " ---> dataByte: ", dataByte, ", result: ", result)
	}

	return method, result, nil
}

func makeCreateContractPairs(method string, abiPath string, pairs []*commonPb.KeyValuePair, runtime commonPb.RuntimeType) (string, []*commonPb.KeyValuePair, error) {
	if runtime != commonPb.RuntimeType_EVM {
		return method, pairs, nil
	}
	data := ""
	//fmt.Println("pairs: ", pairs)

	var params []interface{}
	abiJson, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return method, nil, err
	}
	myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
	if err != nil {
		return method, nil, err
	}

	if printEnable {
		fmt.Println("pairs: ", pairs, ", method: ", method)
	}

	if len(pairs) > 0 {
		var evmAbi []map[string]interface{}
		//abiJson2 := strings.ReplaceAll(abiJson, "\"type\"", "\"typeEvm\"")
		err = json.Unmarshal([]byte(abiJson), &evmAbi)
		if err != nil {
			return method, nil, err
		}

		var methodObj []map[string]interface{}
		for _, obj := range evmAbi {

			if printEnable {
				fmt.Println("obj: ", obj)
			}

			_, ok := obj["name"]
			// 非构造跳过
			if ok {
				continue
			}

			objType, ok := obj["type"]
			// 非构造跳过
			if !ok {
				continue
			}

			//构造函数
			if objType.(string) == "constructor" {
				tmp := obj["inputs"].([]interface{})
				for _, t := range tmp {
					methodObj = append(methodObj, t.(map[string]interface{}))
				}
				break
			}
		}

		fmt.Println("methodObj: ", methodObj)

		for _, paramMap := range methodObj {

			for _, pair := range pairs {

				if printEnable {
					fmt.Println("pair: ", pair, ", paramMap:", paramMap, ",paramMap[\"type\"].(string): ", paramMap["type"].(string), ", paramMap[\"name\"].(string):", paramMap["name"].(string), ", type(pair):", reflect.TypeOf(pair.Value), ", params: ", params)
				}

				if paramMap["name"].(string) == pair.Key {
					switch paramMap["type"].(string) {
					case "address":
						var add *evm.Address
						ski := string(pair.Value)
						fmt.Println("[debug] pair.Key: ", pair.Key, "pair.Value:", pair.Value, ", ski:", ski, ", typeof(ski): ", reflect.TypeOf(ski))
						if strings.LastIndex(string(pair.Value), ".crt") > -1 {
							ski, err = getSki(string(pair.Value))
							if err != nil {
								return method, nil, err
							}
							add, err = getAddr(ski)
							if err != nil {
								return method, nil, err
							}
						} else {
							add, err = getContractAddress(ski)
							fmt.Println("[debug] pair.Key: ", pair.Key, "ski:", ski, ", add: ", add)
							if err != nil {
								return method, nil, err
							}
						}
						params = append(params, add)
						if printEnable {
							fmt.Println("[append][address] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, "type(add):", reflect.TypeOf(add), ", add: ", add, ", params:", params)
						}
					case "uint256", "int256", "uint", "int", "uint8", "int8":
						val, err := strconv.Atoi(string(pair.Value))
						if err != nil {
							return method, nil, err
						}
						params = append(params, big.NewInt(int64(val)))
						if printEnable {
							fmt.Println("[append][uint] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", val:", val, "type(val):", reflect.TypeOf(val), ",big.NewInt(int64(val)): ", big.NewInt(int64(val)), ", params:", params)
						}
					//case "uint", "int", "uint8", "int8":
					//	val, err := strconv.Atoi(pair.Value)
					//	if err != nil {
					//		return method, nil, err
					//	}
					//	params = append(params, big.NewInt(int64(val)))
					case "bool":
						if strings.ToUpper(string(pair.Value)) == "TRUE" {
							params = append(params, true)
						} else {
							params = append(params, false)
						}
						if printEnable {
							fmt.Println("[append][bool] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					case "bytes32", "byte", "bytes1":
						params = append(params, string(pair.Value))
						if printEnable {
							fmt.Println("[append][bytes] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					case "bytes", "string":
						params = append(params, string(pair.Value))
						if printEnable {
							fmt.Println("[append][string] pair.Key: ", pair.Key, "pair.Value: ", pair.Value, ", params:", params)
						}
					}
				}
			}
		}
	}
	if printEnable {
		fmt.Println("change: Pack para: ", method, params)
	}
	dataByte, err := myAbi.Pack(method, params...)
	if printEnable {
		fmt.Println("change: Pack para: ", method, params, " ---> dataByte: ", dataByte, ", err: ", err)
	}
	if err != nil {
		return method, nil, err
	}
	data = hex.EncodeToString(dataByte)
	//fmt.Println("Pack data: ", data)
	if len(data) != 0 {
		method = data[0:8]
	}
	var result []*commonPb.KeyValuePair
	result = []*commonPb.KeyValuePair{
		{
			Key:   "data",
			Value: []byte(data),
		},
	}
	if printEnable {
		fmt.Println("Pack para: ", method, params, " ---> dataByte: ", dataByte, ", result: ", result)
	}

	return method, result, nil
}

func getSki(certPath string) (string, error) {
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("read cert file [%s] failed, %s", certPath, err)
	}

	fmt.Println("certBytes: ", certBytes)

	block, _ := pem.Decode(certBytes)
	cert, err := bcx509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parseCertificate cert failed, %s", err)
	}

	fmt.Println("block: ", block)
	fmt.Println("cert: ", cert)

	ski := hex.EncodeToString(cert.SubjectKeyId)
	fmt.Println("ski: ", ski)
	return ski, nil
}

func getAddr(ski string) (*evm.Address, error) {
	bigInt, err := evm.MakeAddressFromHex(ski)
	addr := evm.BigToAddress(bigInt)

	fmt.Println("bigInt: ", bigInt)
	fmt.Println("addr: ", addr)
	return &addr, err
}

func getContractAddress(name string) (*evm.Address, error) {
	bigInt, err := evm.MakeAddressFromString(name)
	addr := evm.BigToAddress(bigInt)

	fmt.Println("bigInt: ", bigInt)
	fmt.Println("addr: ", addr)
	return &addr, err
}

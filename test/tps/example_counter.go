/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

//
//import (
//	"strconv"
//)
//
////export init
//func init() {
//	LogMessage("[zitao] init test")
//}
//
////export increase
//func increase() {
//	LogMessage("call increase")
//	heartbeatInt := 0
//	if heartbeatString, resultCode := GetState("Counter", "heartbeat"); resultCode != SUCCESS {
//		heartbeatInt = 1
//		LogMessage("call increase, put state from empty")
//		PutState("Counter", "heartbeat", strconv.Itoa(heartbeatInt))
//	} else {
//		heartbeatInt, _ = strconv.Atoi(heartbeatString)
//		heartbeatInt++
//		LogMessage("call increase, put state from exist")
//		PutState("Counter", "heartbeat", strconv.Itoa(heartbeatInt))
//	}
//	SuccessResult(strconv.Itoa(heartbeatInt))
//}
//
////export query
//func query() {
//	LogMessage("call query")
//	if heartbeatString, resultCode := GetState("Counter", "heartbeat"); resultCode != SUCCESS {
//		LogMessage("failed to query state")
//		SuccessResult("0")
//	} else {
//		LogMessage("call query, heartbeat:" + heartbeatString)
//		SuccessResult(heartbeatString)
//	}
//}
//
//type person struct {
//	Name string `json:"name"`
//	Age  int64  `json:"age"`
//}
//
////export for_dag
//func for_dag() {
//	if txId, resultCode := GetTxId(); resultCode != SUCCESS {
//		ErrorResult("failed to get tx id")
//	} else {
//		for i := 0; i < 2; i++ {
//			PutState("test", txId[i:i+1], "value")
//		}
//		SuccessResult("ok")
//	}
//}
//
////export json_ex
//func json_ex() {
//	if num, resultCode := Arg("num"); resultCode != SUCCESS {
//		ErrorResult("failed to get num")
//		return
//	} else {
//		LogMessage("num " + num.(string))
//	}
//	m := Args()
//	LogMessage("m[\"str\"] = " + m["str"].(string))
//	var sth map[string]interface{}
//	sth = m["sth"].(map[string]interface{})
//	LogMessage("m[\"sth\"][\"num1\"] = " + strconv.FormatInt(sth["num1"].(int64), 10))
//	LogMessage("m[\"sth\"][\"num2\"] = " + strconv.FormatInt(sth["num2"].(int64), 10))
//	LogMessage("m[\"sth\"][\"str1\"] = " + sth["str1"].(string))
//	LogMessage("m[\"sth\"][\"str2\"] = " + sth["str2"].(string))
//	var persons []interface{}
//	persons = sth["persons"].([]interface{})
//	var person1 = persons[0].(map[string]interface{})
//	var person2 = persons[1].(map[string]interface{})
//	var p1, p2 person
//	p1.Name = person1["name"].(string)
//	p1.Age = person1["age"].(int64)
//	p2.Name = person2["name"].(string)
//	p2.Age = person2["age"].(int64)
//	LogMessage("person1: " + p1.Name + " age " + strconv.FormatInt(p1.Age, 10))
//	LogMessage("person2: " + p2.Name + " age " + strconv.FormatInt(p2.Age, 10))
//	// _, err := strconv.ParseInt("-1", 10, 64)
//	// if err != nil {
//	// 	LogMessage(err.Error())
//	// }
//	// strconv.FormatInt(1e9, 10)
//}
//
//
//
//
////export calc_json_int64
//func calc_json_int64() {
//	LogMessage("[zitao] input func: calc_json1")
//	calc_json := Args()
//	func_name := calc_json["func_name"].(string)
//	data1 := calc_json["data1"].(string)
//	data2 := calc_json["data2"].(string)
//	data3 := calc_json["data3"].(string)
//	LogMessage("[zitao] calc_json[func_name]: " + func_name)
//	LogMessage("[zitao] calc_json[data1]: " + data1)
//	LogMessage("[zitao] calc_json[data2]: " + data2)
//	LogMessage("[zitao] calc_json[data3]: " + data3)
//
//	idata1, _ := strconv.ParseInt(data1, 10, 64)
//	idata2, _ := strconv.ParseInt(data2, 10, 64)
//	var result_str string
//	var result int64
//	status := false
//	if func_name == "add" {
//		result = idata1 + idata2
//		result_str = strconv.FormatInt(result, 10)
//		status = true
//	}else if func_name == "sub" {
//		result = idata1 - idata2
//		result_str = strconv.FormatInt(result, 10)
//		status = true
//	}else if func_name == "mul" {
//		result = idata1 * idata2
//		result_str = strconv.FormatInt(result, 10)
//		status = true
//	}else if func_name == "div" {
//		result = idata1 / idata2
//		result_str = strconv.FormatInt(result, 10)
//		status = true
//	}else if func_name == "set_data" {
//		result_str = data3
//		status = true
//	}else if func_name == "failure" {
//		LogMessage("[zitao] calc_json[func_name] failure result: " + func_name)
//		ErrorResult("zitao test error")
//	}else if func_name == "delete" {
//		LogMessage("[zitao] calc_json[func_name] DeleteState result: " + func_name)
//		DeleteState("zitao", func_name)
//	}else {
//		LogMessage("[zitao] panic test")
//		panic("zitao test panic !!!")
//	}
//	if status {
//		PutState("zitao", func_name, result_str)
//		LogMessage("[zitao] calc_json[func_name] result: " + result_str)
//	}
//	SuccessResult("ok")
//}
//
////export calc_json
//func calc_json() {
//	LogMessage("[zitao] input func: calc_json")
//	calc_json := Args()
//	func_name := calc_json["func_name"].(string)
//	data1 := calc_json["data1"].(string)
//	data2 := calc_json["data2"].(string)
//	LogMessage("[zitao] calc_json[func_name]: " + func_name)
//	LogMessage("[zitao] calc_json[data1]: " + data1)
//	LogMessage("[zitao] calc_json[data2]: " + data2)
//
//	idata1, _ := strconv.Atoi(data1)
//	idata2, _ := strconv.Atoi(data2)
//	LogMessage("[zitao] change calc_json[data1]: " + strconv.FormatInt(int64(idata1), 10))
//	LogMessage("[zitao] change calc_json[data2]: " + strconv.FormatInt(int64(idata2), 10))
//	var result_str string
//	var result int
//	status := false
//	if func_name == "add" {
//		result = idata1 + idata2
//		result_str = strconv.FormatInt(int64(result), 10)
//		status = true
//	} else if func_name == "sub" {
//		result = idata1 - idata2
//		result_str = strconv.FormatInt(int64(result), 10)
//		status = true
//	} else if func_name == "mul" {
//		result = idata1 * idata2
//		result_str = strconv.FormatInt(int64(result), 10)
//		status = true
//	} else if func_name == "div" {
//		result = idata1 / idata2
//		result_str = strconv.FormatInt(int64(result), 10)
//		status = true
//	} else if func_name == "set_data" {
//		data3 := calc_json["data3"].(string)
//		data4 := calc_json["data4"].(string)
//		LogMessage("[zitao] calc_json[data3]: " + data3)
//		LogMessage("[zitao] calc_json[data4]: " + data4)
//		PutState("zitao", data3, data4)
//		LogMessage("[zitao] calc_json[func_name] result: " + result_str)
//		status = true
//	}else if func_name == "failure" {
//		data3 := calc_json["data3"].(string)
//		LogMessage("[zitao] calc_json[data3]: " + data3)
//		LogMessage("[zitao] calc_json[func_name] failure set result: " + data3)
//		PutState("zitao", func_name, data3)
//		ErrorResult("zitao test error")
//	} else if func_name == "delete" {
//		data3 := calc_json["data3"].(string)
//		LogMessage("[zitao] calc_json[data3]: " + data3)
//		LogMessage("[zitao] calc_json[func_name] delete name: " + data3)
//		DeleteState("zitao", data3)
//		status = true
//	} else {
//		LogMessage("[zitao] panic test")
//		PutState("zitao", func_name, "panic")
//		panic("zitao test panic !!!")
//	}
//	if status {
//		PutState("zitao", func_name, result_str)
//		LogMessage("[zitao] calc_json[func_name] result: " + result_str)
//		SuccessResult("ok")
//	}
//}
//
////export get_calc
//func get_calc() {
//	LogMessage("[zitao] input func: get_json")
//	func_name, resultCode := Arg("func_name"); if resultCode != SUCCESS {
//		ErrorResult("failure get func_name")
//		return
//	}
//	LogMessage("[zitao] get_calc[func_name]: " + func_name.(string))
//
//	result, _ := GetState("zitao", func_name.(string))
//	LogMessage("[zitao] calc_json[func_name] result: " + result)
//	SuccessResult(result)
//}
//
//
////export call_self
//func call_self() {
//	LogMessage("[zitao] input func: call_self")
//
//	callnum_str, _ := GetState("zitao", "callnum")
//	icallnum, _ := strconv.Atoi(callnum_str)
//	LogMessage("[zitao] change calc_json[callnum]: " + strconv.FormatInt(int64(icallnum), 10))
//	icallnum = icallnum - 1
//	PutState("zitao", "callnum", strconv.FormatInt(int64(icallnum), 10))
//	if icallnum < 1 {
//		LogMessage("[zitao] call_self[callnum] result(end): " + strconv.FormatInt(int64(icallnum), 10))
//		SuccessResult("finish call_self")
//	} else{
//		LogMessage("[zitao] call_self[callnum] result(test): " + strconv.FormatInt(int64(icallnum), 10))
//		call_self()
//	}
//}
//
//
////export loop_test
//func loop_test() {
//	LogMessage("[zitao] input func: loop_test")
//
//	loopnum_str, _ := GetState("zitao", "loopnum")
//	iloopnum, _ := strconv.Atoi(loopnum_str)
//	LogMessage("[zitao] change loop_test[loopnum]: " + strconv.FormatInt(int64(iloopnum), 10))
//	for i := iloopnum; i > 1; i-- {
//		LogMessage("[zitao] change loop_test[i]: " + strconv.FormatInt(int64(i), 10))
//		PutState("zitao", "loopnum", strconv.FormatInt(int64(i), 10))
//	}
//
//	SuccessResult("finish loop_test")
//}
//
//
////export set_store
//func set_store() {
//	LogMessage("[zitao] ========================================start")
//	LogMessage("[zitao] input func: set_store")
//	set_store_params := Args()
//	key := set_store_params["key"].(string)
//	name := set_store_params["name"].(string)
//	value := set_store_params["value"].(string)
//	LogMessage("[zitao] change set_store[key]: " + key)
//	LogMessage("[zitao] change set_store[name]: " + name)
//	LogMessage("[zitao] change set_store[value]: " + value)
//	result := PutState(key, name, value)
//	LogMessage("[zitao] PutState: key=" + key + ",name=" + name + ",value=" + value + ",result:" + strconv.FormatInt(int64(result), 10))
//	LogMessage("[zitao] ========================================end")
//	SuccessResult("finish set_store")
//}
//
////export get_store
//func get_store() {
//	LogMessage("[zitao] ========================================start")
//	LogMessage("[zitao] input func: get_store")
//	set_store_params := Args()
//	key := set_store_params["key"].(string)
//	name := set_store_params["name"].(string)
//	LogMessage("[zitao] change get_store[key]: " + key)
//	LogMessage("[zitao] change get_store[name]: " + name)
//	value, result := GetState(key, name)
//	LogMessage("[zitao] GetState: key=" + key + ",name=" + name + ",result:" + strconv.FormatInt(int64(result), 10))
//	LogMessage("[zitao] change get_store[value]: " + value)
//	LogMessage("[zitao] ========================================end")
//	SuccessResult(value)
//}
//
////export delete_store
//func delete_store() {
//	LogMessage("[zitao] ========================================start")
//	LogMessage("[zitao] input func: delete_store")
//	set_store_params := Args()
//	key := set_store_params["key"].(string)
//	name := set_store_params["name"].(string)
//	LogMessage("[zitao] change delete_store[key]: " + key)
//	LogMessage("[zitao] change delete_store[name]: " + name)
//	result := DeleteState(key, name)
//	LogMessage("[zitao] DeleteState: key=" + key + ",name=" + name + ",result:" + strconv.FormatInt(int64(result), 10))
//	LogMessage("[zitao] ========================================end")
//	SuccessResult("finish delete_store")
//}
//
//func main() {
//
//}

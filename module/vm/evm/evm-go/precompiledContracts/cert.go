/*
 * Copyright 2020 The SealEVM Authors
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package precompiledContracts

import "chainmaker.org/chainmaker-go/evm/evm-go/params"

type senderOrgId struct {
	//Value string
}

//func (o senderOrgId) SetValue(v string) {
//	o.Value = v
//}
func (o senderOrgId) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o senderOrgId) Execute(input []byte) ([]byte, error) {
	return input, nil
}

type senderRole struct {
	//Value string
}

//func (o senderRole) SetValue(v string) {
//	o.Value = v
//}

func (o senderRole) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o senderRole) Execute(input []byte) ([]byte, error) {
	return input, nil
}

type senderPk struct {
	//Value string
}

//func (o senderPk) SetValue(v string) {
//	o.Value = v
//}

func (o senderPk) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o senderPk) Execute(input []byte) ([]byte, error) {
	return input, nil
}

type creatorOrgId struct {
	//Value string
}

//func (o creatorOrgId) SetValue(v string) {
//	o.Value = v
//}

func (o creatorOrgId) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o creatorOrgId) Execute(input []byte) ([]byte, error) {
	return input, nil
}

type creatorRole struct {
	//Value string
}

//func (o creatorRole) SetValue(v string) {
//	o.Value = v
//}

func (o creatorRole) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o creatorRole) Execute(input []byte) ([]byte, error) {
	return input, nil
}

type creatorPk struct {
	//Value string
}

//func (o creatorPk) SetValue(v string) {
//	o.Value = v
//}

func (o creatorPk) GasCost(input []byte) uint64 {
	return params.EcrecoverGas
}

func (o creatorPk) Execute(input []byte) ([]byte, error) {
	return input, nil
}

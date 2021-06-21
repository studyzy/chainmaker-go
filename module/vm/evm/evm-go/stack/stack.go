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

package stack

import (
	"chainmaker.org/chainmaker/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go/utils"
)

type Stack struct {
	data []*evmutils.Int
	max  int
}

func New(max int) *Stack {
	if max <= 0 {
		max = int(^uint(0) >> 1)
	}

	return &Stack{
		max: max,
	}
}

func (s *Stack) GetData() []*evmutils.Int {
	return s.data
}
func (s *Stack) CheckStackDepth(minRequire int, willAdd int) error {
	sLen := len(s.data)
	if sLen < minRequire {
		return utils.ErrStackUnderFlow
	} else if sLen+willAdd > s.max {
		return utils.ErrStackOverFlow
	}

	return nil
}

func (s *Stack) Len() int {
	return len(s.data)
}

func (s *Stack) Push(i *evmutils.Int) {
	s.data = append(s.data, i)
	return
}

func (s *Stack) PushN(i []*evmutils.Int) {
	s.data = append(s.data, i...)
	return
}

func (s *Stack) Pop() *evmutils.Int {
	sLen := len(s.data)
	i := s.data[sLen-1]
	s.data = s.data[:sLen-1]
	return i
}

func (s *Stack) PopN(n int) []*evmutils.Int {
	sLen := len(s.data)
	var el []*evmutils.Int
	el = s.data[sLen-n:]
	s.data = s.data[:sLen-n]

	//reverse to make sure the order
	for i, j := 0, len(el)-1; i < j; i, j = i+1, j-1 {
		el[i], el[j] = el[j], el[i]
	}
	return el
}

func (s *Stack) Peek() *evmutils.Int {
	sLen := len(s.data)
	if sLen == 0 {
		return nil
	}

	i := s.data[sLen-1]
	return i
}

func (s *Stack) PeekN(n int) []*evmutils.Int {
	sLen := len(s.data)
	var el []*evmutils.Int = nil
	if sLen >= n {
		el = s.data[sLen-n:]
	}

	return el
}

func (s *Stack) Swap(n int) {
	n += 1
	sLen := len(s.data)

	s.data[sLen-n], s.data[sLen-1] = s.data[sLen-1], s.data[sLen-n]

	return
}

func (s *Stack) Dup(n int) {
	sLen := len(s.data)

	i := s.data[sLen-n]
	newI := evmutils.FromBigInt(i.Int)
	s.Push(newI)

	return
}

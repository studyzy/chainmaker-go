// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

func SearchInt64(n int64, f func(int64) (bool, error)) (int64, error) {
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := int64(0), n
	for i < j {
		h := int64(uint64(i+j) >> 1) // avoid overflow when computing h

		ok, err := f(h)
		if err != nil {
			return -1, err
		}
		// i ≤ h < j
		if !ok {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i, nil
}

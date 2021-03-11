/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import "fmt"

func main() {
	if err := testASym(); err != nil {
		fmt.Println(err)
		return
	}
	if err := testSym(); err != nil {
		fmt.Println(err)
		return
	}
	testHash()

	fmt.Println("\ntest all finished.")
}

/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/utils/v2"
	"github.com/mr-tron/base58/base58"
)

var (
	certPath = ""
)

func main() {
	flag.StringVar(&certPath, "cert_path", "", "path of cert that will calculate address")
	flag.Parse()
	calAddressFromCert()
}

func calAddressFromCert() {
	if len(certPath) == 0 {
		panic("cert path is null")
	}

	certContent, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(fmt.Errorf("read cert content failed, reason: %s", err))
	}
	cert, err := utils.ParseCert(certContent)
	if err != nil {
		panic(fmt.Errorf("parse cert failed, reason: %s", err))
	}
	pubkey, err := cert.PublicKey.Bytes()
	if err != nil {
		panic(fmt.Errorf("get pubkey failed from cert, reason: %s", err))
	}
	hash := sha256.Sum256(pubkey)
	addr := base58.Encode(hash[:])
	fmt.Printf("address: %s from cert: %s\n", addr, certPath)
}

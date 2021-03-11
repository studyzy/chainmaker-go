/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"chainmaker.org/chainmaker-go/protocol"
)

// structure to store cached signers: speed up verification, support CRL
type cachedSigner struct {
	signer    protocol.Member
	certChain []*bcx509.Certificate
}

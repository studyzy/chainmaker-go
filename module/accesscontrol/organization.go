/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
)

type organization struct {
	// Name of this group
	id string

	// Trusted certificates or white list
	trustedRootCerts map[string]*bcx509.Certificate

	// Trusted intermediate certificates or white list
	trustedIntermediateCerts map[string]*bcx509.Certificate
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sdkPbCommon "chainmaker.org/chainmaker/pb-go/common"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestSignConfigUpdatePayload(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "cmc")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	generateConfigUpdatePayload(t, tmpDir)

	signOutput = filepath.Join(tmpDir, "config_collect-signed.pb")
	signInput = filepath.Join(tmpDir, "config_collect.pb")

	orgId = "wx-org1.chainmaker.org"
	adminKeyPath = "../../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key"
	adminCertPath = "../../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"

	err = signPayload()
	assert.NoError(t, err)

	raw, err := ioutil.ReadFile(signOutput)
	assert.NoError(t, err)

	payload := &sdkPbCommon.TxRequest{}
	err = proto.Unmarshal(raw, payload)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(payload.Endorsers))
	assert.NotNil(t, payload.Endorsers)
	assert.NotNil(t, payload.Endorsers[0])
}

func TestSignContractMgmtPayload(t *testing.T) {
	//tmpDir, err := ioutil.TempDir("", "cmc")
	//assert.NoError(t, err)
	//defer os.RemoveAll(tmpDir)
	//
	//generateContractMgmtPayload(t, tmpDir)
	//
	//signOutput = filepath.Join(tmpDir, "contract_collect-signed.pb")
	//signInput = filepath.Join(tmpDir, "contract_collect.pb")
	//
	//orgId = "wx-org1.chainmaker.org"
	//adminKeyPath = "../../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key"
	//adminCertPath = "../../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
	//
	//err = signContractMgmtPayload()
	//assert.NoError(t, err)
	//
	//raw, err := ioutil.ReadFile(signOutput)
	//assert.NoError(t, err)
	//
	//payload := &sdkPbCommon.Payload{}
	//err = proto.Unmarshal(raw, payload)
	//assert.NoError(t, err)
	//
	//assert.Equal(t, 1, len(payload.Endorsement))
	//assert.NotNil(t, payload.Endorsement)
	//assert.NotNil(t, payload.Endorsement[0])
}

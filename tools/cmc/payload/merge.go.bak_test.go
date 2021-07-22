/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sdkPbCommon "chainmaker.org/chainmaker/pb-go/common"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestMergeConfigUpdatePayload(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "cmc")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	generateConfigUpdatePayload(t, tmpDir)
	generateConfigUpdatePayloads(t, tmpDir, 4)

	mergeOutput = filepath.Join(tmpDir, "config_collect-signed-all.pb")
	mergeInputs = []string{
		filepath.Join(tmpDir, "config_collect-signed1.pb"),
		filepath.Join(tmpDir, "config_collect-signed2.pb"),
		filepath.Join(tmpDir, "config_collect-signed3.pb"),
		filepath.Join(tmpDir, "config_collect-signed4.pb"),
	}

	err = mergeSystemContractPayload()
	assert.NoError(t, err)

	raw, err := ioutil.ReadFile(mergeOutput)
	assert.NoError(t, err)

	payload := &sdkPbCommon.Payload{}
	err = proto.Unmarshal(raw, payload)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(payload.Endorsement))
	for _, endorsement := range payload.Endorsement {
		assert.NotNil(t, endorsement)
	}
}

func TestMergeContractMgmtPayload(t *testing.T) {
	//tmpDir, err := ioutil.TempDir("", "cmc")
	//assert.NoError(t, err)
	//defer os.RemoveAll(tmpDir)
	//
	//generateContractMgmtPayload(t, tmpDir)
	//generateContractMgmtPayloads(t, tmpDir, 4)
	//
	//mergeOutput = filepath.Join(tmpDir, "contract_collect-signed-all.pb")
	//mergeInputs = []string{
	//	filepath.Join(tmpDir, "contract_collect-signed1.pb"),
	//	filepath.Join(tmpDir, "contract_collect-signed2.pb"),
	//	filepath.Join(tmpDir, "contract_collect-signed3.pb"),
	//	filepath.Join(tmpDir, "contract_collect-signed4.pb"),
	//}
	//
	//err = mergeContractMgmtPayload()
	//assert.NoError(t, err)
	//
	//raw, err := ioutil.ReadFile(mergeOutput)
	//assert.NoError(t, err)
	//
	//payload := &sdkPbCommon.Payload{}
	//err = proto.Unmarshal(raw, payload)
	//assert.NoError(t, err)
	//
	//assert.Equal(t, 4, len(payload.Endorsement))
	//for _, endorsement := range payload.Endorsement {
	//	assert.NotNil(t, endorsement)
	//}
}

func generateConfigUpdatePayloads(t *testing.T, tmpDir string, len int) {
	for i := 1; i <= len; i++ {
		signOutput = filepath.Join(tmpDir, fmt.Sprintf("config_collect-signed%d.pb", i))
		signInput = filepath.Join(tmpDir, "config_collect.pb")

		orgId = fmt.Sprintf("wx-org%d.chainmaker.org", i)
		adminKeyPath = fmt.Sprintf("../../../config/crypto-config/wx-org%d.chainmaker.org/user/admin1/admin1.sign.key", i)
		adminCertPath = fmt.Sprintf("../../../config/crypto-config/wx-org%d.chainmaker.org/user/admin1/admin1.sign.crt", i)

		err := signSystemContractPayload()
		assert.NoError(t, err)
	}
}

func generateContractMgmtPayloads(t *testing.T, tmpDir string, len int) {
	for i := 1; i <= len; i++ {
		signOutput = filepath.Join(tmpDir, fmt.Sprintf("contract_collect-signed%d.pb", i))
		signInput = filepath.Join(tmpDir, "contract_collect.pb")

		orgId = fmt.Sprintf("wx-org%d.chainmaker.org", i)
		adminKeyPath = fmt.Sprintf("../../../config/crypto-config/wx-org%d.chainmaker.org/user/admin1/admin1.sign.key", i)
		adminCertPath = fmt.Sprintf("../../../config/crypto-config/wx-org%d.chainmaker.org/user/admin1/admin1.sign.crt", i)

		err := signContractMgmtPayload()
		assert.NoError(t, err)
	}
}

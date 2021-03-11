/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

import "testing"

func TestLogger(_ *testing.T) {
	logger := GetLogger(MODULE_CORE)
	logger.Infof("core log ......")

	logger = GetLogger(MODULE_CONSENSUS)
	logger.Infof("consensus log .....")

	logger = GetLogger(MODULE_EVENT)
	logger.Infof("event log .....")

	logger = GetLogger(MODULE_BRIEF)
	logger.Infof("brief log .....")
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localconf

var (
	//CurrentVersion current version
	CurrentVersion = "v2.0.2"
	// CurrentCommit current git commit hash
	GitCommit = ""
	// CurrentBranch current git branch
	GitBranch = ""
	// BuildDateTime compile datetime
	BuildDateTime = ""
)

var (
	//ConfigFilepath 配置文件的路径，默认为当前文件夹的chainmaker.yml文件
	ConfigFilepath = "./chainmaker.yml"
)

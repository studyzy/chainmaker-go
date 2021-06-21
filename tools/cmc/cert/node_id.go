/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/common/helper"
	"github.com/spf13/cobra"
)

func nodeIdCMD() *cobra.Command {
	nodeIdCmd := &cobra.Command{
		Use:   "nid",
		Short: "get node id",
		Long:  "Get node id of node cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getNodeId()
		},
	}

	attachFlags(nodeIdCmd, []string{
		flagNodeCertPath,
	})

	return nodeIdCmd
}

func getNodeId() error {
	file, err := ioutil.ReadFile(nodeCertPath)
	if err != nil {
		return err
	}
	nodeId, err := helper.GetLibp2pPeerIdFromCert(file)
	if err != nil {
		return err
	}
	fmt.Printf("node id : %s \n", nodeId)
	return nil
}

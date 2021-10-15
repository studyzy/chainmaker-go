/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
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
		flagNodeCertPath, flagNodePkPath,
	})

	return nodeIdCmd
}

func getNodeId() error {
	if nodeCertPath != "" {
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
	} else if nodePkPath != "" {
		file, err := ioutil.ReadFile(nodePkPath)
		if err != nil {
			return err
		}
		pk, err := asym.PublicKeyFromPEM([]byte(file))
		if err != nil {
			return err
		}
		nodeId, err := helper.CreateLibp2pPeerIdWithPublicKey(pk)
		if err != nil {
			return err
		}
		fmt.Printf("node id : %s \n", nodeId)
		return nil
	} else {
		fmt.Printf("invalid parameter\n")
		return nil
	}

}

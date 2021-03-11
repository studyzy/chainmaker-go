/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

// TBFT chain config keys
const (
	TBFT_propose_timeout_key       = "TBFT_propose_timeout"
	TBFT_propose_delta_timeout_key = "TBFT_propose_delta_timeout"
	TBFT_blocks_per_proposer       = "TBFT_blocks_per_proposer"
)

// TBFT data key in Block.AdditionalData.ExtraData
const (
	TBFTAddtionalDataKey = "TBFTAddtionalDataKey"
	RAFTAddtionalDataKey = "RAFTAddtionalDataKey"
)

type ConsensusEngine interface {
	// Init starts the consensus engine.
	Start() error

	// Stop stops the consensus engine.
	Stop() error
}

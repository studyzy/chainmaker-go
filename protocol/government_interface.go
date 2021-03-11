/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

type Government interface {
	//used to verify consensus data
	Verifier
	//get current epoch id
	GetEpochId() uint64
	//get number of validators
	GetGovMembersValidatorCount() uint64
	//get min alive validators number
	GetGovMembersValidatorMinCount() uint64
	//used to specify MBFT how many recent blocks in cache. The default is 0, which means no cache
	GetCachedLen() uint64
	//get current nodes
	GetMembers() interface{}
	//get current epoch's validators
	GetValidators() interface{}
	//get next epoch's validators
	GetNextValidators() interface{}
	//get the block height of switching next epoch
	GetSwitchHeight() uint64
	//get the time, which means the timeout period for starting the next round of block consensus after sealblock
	GetSkipTimeoutCommit() bool
	//get validator continuous propose count,  used to validator switching
	GetNodeProposeRound() uint64
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/consensus/government"

	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	"chainmaker.org/chainmaker-go/protocol"
	chainUtils "chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
)

//GetConsensusArgsFromBlock get args from block
func GetConsensusArgsFromBlock(block *common.Block) (*consensus.BlockHeaderConsensusArgs, error) {
	if block == nil {
		return nil, nil
	}
	args := new(consensus.BlockHeaderConsensusArgs)
	if block.Header.ConsensusArgs == nil {
		return nil, nil
	}
	err := proto.Unmarshal(block.Header.ConsensusArgs, args)
	if err != nil {
		return nil, err
	}
	return args, nil
}

//GetLevelFromBlock get level from block
func GetLevelFromBlock(block *common.Block) (uint64, error) {
	args, err := GetConsensusArgsFromBlock(block)
	if err != nil || args == nil {
		return 0, err
	}
	return uint64(args.Level), nil
}

//GetQCFromBlock get qc from block
func GetQCFromBlock(block *common.Block) []byte {
	var qc []byte = nil
	if block == nil || block.AdditionalData == nil || block.AdditionalData.ExtraData == nil {
		return nil
	}
	if v, ok := block.AdditionalData.ExtraData["QC"]; ok {
		qc = v
	}
	return qc
}

//GetLevelFromQc get level from qc
func GetLevelFromQc(block *common.Block) (uint64, error) {
	qc := new(chainedbftpb.QuorumCert)
	err := proto.Unmarshal(GetQCFromBlock(block), qc)
	if err != nil || qc == nil {
		return 0, err
	}
	return qc.Level, nil
}

//AddQCtoBlock add qc to block
func AddQCtoBlock(block *common.Block, qc []byte) error {
	if block == nil {
		return nil
	}
	if block.AdditionalData == nil {
		block.AdditionalData = &common.AdditionalData{
			ExtraData: make(map[string][]byte),
		}
	}
	if block.AdditionalData.ExtraData == nil {
		block.AdditionalData.ExtraData = make(map[string][]byte)
	}
	block.AdditionalData.ExtraData["QC"] = qc

	return nil
}

//SignBlock signs the block using given key
func SignBlock(block *common.Block, hashType string, signer protocol.SigningMember) error {
	hash, sig, err := chainUtils.SignBlock(hashType, signer, block)
	if err != nil {
		return err
	}
	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig

	return nil
}

//SignConsensusMsg signs the consensus msg using given key
func SignConsensusMsg(msg *chainedbftpb.ConsensusMsg, hashType string,
	signer protocol.SigningMember) error {
	if msg.Payload == nil {
		return fmt.Errorf("msg payload is nil")
	}
	data, err := proto.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed, payload %v, err %v", msg.Payload, err)
	}

	sign, err := signer.Sign(hashType, data)
	if err != nil {
		return fmt.Errorf("sign data failed, err %v data %v", err, data)
	}
	serializeMember, err := signer.GetSerializedMember(true)
	if err != nil {
		return fmt.Errorf("get signer serializeMember failed, err %v", err)
	}

	msg.SignEntry = &common.EndorsementEntry{
		Signer:    serializeMember,
		Signature: sign,
	}
	return nil
}

//AddConsensusArgstoBlock add consensus args to block
func AddConsensusArgstoBlock(block *common.Block, level uint64, txRWSet *common.TxRWSet) error {
	if block == nil {
		return nil
	}
	consensusArgs := &consensus.BlockHeaderConsensusArgs{
		ConsensusType: int64(consensus.ConsensusType_HOTSTUFF),
		Level:         level,
		ConsensusData: txRWSet,
	}
	argBytes, err := proto.Marshal(consensusArgs)
	if err != nil {
		return err
	}
	block.Header.ConsensusArgs = argBytes
	return nil
}

//VerifyConsensusMsgSign verify the consensus msg sign
func VerifyConsensusMsgSign(msg *chainedbftpb.ConsensusMsg, ac protocol.AccessControlProvider) error {
	if msg.Payload == nil {
		return fmt.Errorf("msg payload is nil")
	}
	data, err := proto.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed, payload %v , err %v", msg.Payload, err)
	}

	return VerifyDataSign(data, msg.SignEntry, ac)
}

//VerifyDataSign verify the data with EndorsementEntry, ac, org
func VerifyDataSign(data []byte, signEnrty *common.EndorsementEntry,
	ac protocol.AccessControlProvider) error {

	principal, err := ac.CreatePrincipal(
		protocol.ResourceNameConsensusNode,
		[]*common.EndorsementEntry{signEnrty},
		data,
	)
	if err != nil {
		return fmt.Errorf("new principal error %v", err)
	}

	result, err := ac.VerifyPrincipal(principal)
	if err != nil {
		return fmt.Errorf("verify principal failed, error %v, data %v", err, data)
	}
	if !result {
		return fmt.Errorf("verify failed, result %v, data %v", result, data)
	}

	return nil
}

//GetUidFromProtoSigner get uid from SerializedMember using netservice
func GetUidFromProtoSigner(signerpb *pbac.SerializedMember, netservice protocol.NetService, ac protocol.AccessControlProvider) (string, error) {
	if signerpb == nil {
		return "", fmt.Errorf("signer is nil")
	}
	member, err := ac.NewMemberFromProto(signerpb)
	if err != nil {
		return "", fmt.Errorf("new member from proto failed, err: %v", err)
	}

	certId := member.GetMemberId()
	uid, err := netservice.GetNodeUidByCertId(certId)
	if err != nil {
		return "", fmt.Errorf("get node uid by certid failed, err: %v", err)
	}
	return uid, nil
}

func GetEnableEpochFlagFromConf(chainConf protocol.ChainConf) (enable bool) {
	config := chainConf.ChainConfig()
	for _, kv := range config.Consensus.ExtConfig {
		if kv.Key == government.EnableEpoch {
			return strings.ToUpper(kv.Value) == "TRUE"
		}
	}
	return false
}

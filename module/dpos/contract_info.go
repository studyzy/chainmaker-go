package dpos

import (
	"encoding/binary"
	"math/big"

	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	dpospb "chainmaker.org/chainmaker-go/pb/protogo/dpos"

	"github.com/golang/protobuf/proto"
)

// getEpochInfo get epoch info from ledger
func (impl *DposImpl) getEpochInfo() (*commonpb.Epoch, error) {
	val, err := impl.stateDB.ReadObject(commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(), []byte(commonpb.StakePrefix_Prefix_Curr_Epoch.String()))
	if err != nil {
		impl.log.Errorf("read contract: %s error: %s", commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(), err)
		return nil, err
	}

	epoch := commonpb.Epoch{}
	if err = proto.Unmarshal(val, &epoch); err != nil {
		impl.log.Errorf("unmarshal epoch failed, reason: %s", err)
		return nil, err
	}
	return &epoch, nil
}

// getAllCandidateInfo get all candidates from ledger
func (impl *DposImpl) getAllCandidateInfo() ([]*dpospb.CandidateInfo, error) {
	preFix := []byte(commonpb.StakePrefix_Prefix_Validator.String())
	iter, err := impl.stateDB.SelectObject(commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(), preFix, BytesPrefix(preFix))
	if err != nil {
		impl.log.Errorf("read contract: %s error: %s", commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(), err)
		return nil, err
	}
	defer iter.Release()

	vals := make([]*commonpb.Validator, 0, 10)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			impl.log.Errorf("iterator read error: %s", err)
			return nil, err
		}
		val := commonpb.Validator{}
		if err = proto.Unmarshal(kv.Value, &val); err != nil {
			impl.log.Errorf("unmarshal validator failed, reason: %s", err)
			return nil, err
		}
		vals = append(vals, &val)
	}
	if len(vals) == 0 {
		impl.log.Warnf("not find candidate .")
		return nil, nil
	}
	candidates := make([]*dpospb.CandidateInfo, 0, len(vals))
	for i := 0; i < len(vals); i++ {
		if !vals[i].Jailed && vals[i].Status == commonpb.BondStatus_Bonded {
			candidates = append(candidates, &dpospb.CandidateInfo{
				PeerID: vals[i].ValidatorAddress,
				Weight: vals[i].Tokens,
			})
		}
	}
	return candidates, nil
}

func (impl *DposImpl) createEpochRwSet(epoch *commonpb.Epoch) (*commonpb.TxRWSet, error) {
	id := make([]byte, 8)
	currPreFix := []byte(commonpb.StakePrefix_Prefix_Curr_Epoch.String())
	recordPreFix := []byte(commonpb.StakePrefix_Prefix_Epoch_Record.String())

	binary.BigEndian.PutUint64(id, epoch.EpochID)
	bz, err := proto.Marshal(epoch)
	if err != nil {
		impl.log.Errorf("marshal epoch failed, reason: %s", err)
		return nil, err
	}

	rw := &commonpb.TxRWSet{
		TxId: "",
		TxWrites: []*commonpb.TxWrite{
			{
				ContractName: commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(),
				Key:          currPreFix,
				Value:        bz,
			},
			{
				ContractName: commonpb.ContractName_SYSTEM_CONTRACT_STATE.String(),
				Key:          append(recordPreFix, id...),
				Value:        bz,
			},
		},
	}
	return rw, nil
}

func (impl *DposImpl) createRewardRwSet(rewardAmount big.Int) (*commonpb.TxRWSet, error) {
	return nil, nil
}

func (impl *DposImpl) createSlashRwSet(slashAmount big.Int) (*commonpb.TxRWSet, error) {
	return nil, nil
}

// BytesPrefix returns key range that satisfy the given prefix.
// This only applicable for the standard 'bytes comparer'.
func BytesPrefix(start []byte) (end []byte) {
	var limit []byte
	for i := len(start) - 1; i >= 0; i-- {
		c := start[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, start)
			limit[i] = c + 1
			break
		}
	}
	return limit
}

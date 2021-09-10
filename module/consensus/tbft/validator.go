/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"chainmaker.org/chainmaker/logger/v2"
)

var (
	ErrInvalidIndex = errors.New("invalid index")
)

type validatorSet struct {
	sync.Mutex
	logger            *logger.CMLogger
	Validators        []string
	blocksPerProposer uint64
}

func newValidatorSet(logger *logger.CMLogger, validators []string, blocksPerProposer uint64) *validatorSet {
	sort.SliceStable(validators, func(i, j int) bool {
		return validators[i] < validators[j]
	})

	valSet := &validatorSet{
		logger:            logger,
		Validators:        validators,
		blocksPerProposer: blocksPerProposer,
	}
	valSet.logger.Infof("new validator set: %v", validators)

	return valSet
}

func (valSet *validatorSet) isNilOrEmpty() bool {
	if valSet == nil {
		return true
	}
	valSet.Lock()
	defer valSet.Unlock()
	return len(valSet.Validators) == 0
}

func (valSet *validatorSet) String() string {
	if valSet == nil {
		return ""
	}
	valSet.Lock()
	defer valSet.Unlock()

	return fmt.Sprintf("%v", valSet.Validators)

}

func (valSet *validatorSet) Size() int32 {
	if valSet == nil {
		return 0
	}

	valSet.Lock()
	defer valSet.Unlock()

	return int32(len(valSet.Validators))
}

// HasValidator holds the lock and return whether validator is in
// the validatorSet
func (valSet *validatorSet) HasValidator(validator string) bool {
	if valSet == nil {
		return false
	}

	valSet.Lock()
	defer valSet.Unlock()

	return valSet.hasValidator(validator)
}

func (valSet *validatorSet) hasValidator(validator string) bool {
	for _, val := range valSet.Validators {
		if val == validator {
			return true
		}
	}
	return false
}

func (valSet *validatorSet) GetProposer(height uint64, round int32) (validator string, err error) {
	if valSet.isNilOrEmpty() {
		return "", ErrInvalidIndex
	}

	heightOffset := int32((height + 1) / valSet.blocksPerProposer)
	roundOffset := round % valSet.Size()
	proposerIndex := (heightOffset + roundOffset) % valSet.Size()

	return valSet.getByIndex(proposerIndex)
}

func (valSet *validatorSet) updateValidators(validators []string) (addedValidators []string, removedValidators []string,
	err error) {
	valSet.Lock()
	defer valSet.Unlock()

	removedValidatorsMap := make(map[string]bool)
	for _, v := range valSet.Validators {
		removedValidatorsMap[v] = true
	}

	for _, v := range validators {
		// addedValidators
		if !valSet.hasValidator(v) {
			addedValidators = append(addedValidators, v)
		}

		delete(removedValidatorsMap, v)
	}

	// removedValidators
	for k := range removedValidatorsMap {
		removedValidators = append(removedValidators, k)
	}

	sort.SliceStable(validators, func(i, j int) bool {
		return validators[i] < validators[j]
	})

	valSet.Validators = validators

	sort.SliceStable(addedValidators, func(i, j int) bool {
		return addedValidators[i] < addedValidators[j]
	})

	sort.SliceStable(removedValidators, func(i, j int) bool {
		return removedValidators[i] < removedValidators[j]
	})
	valSet.logger.Infof("%v update validators, validators: %v, addedValidators: %v, removedValidators: %v",
		valSet.Validators, validators, addedValidators, removedValidators)
	return
}

func (valSet *validatorSet) updateBlocksPerProposer(blocks uint64) error {
	valSet.Lock()
	defer valSet.Unlock()

	valSet.blocksPerProposer = blocks

	return nil
}

func (valSet *validatorSet) getByIndex(index int32) (validator string, err error) {
	if index < 0 || index >= valSet.Size() {
		return "", ErrInvalidIndex
	}

	valSet.Lock()
	defer valSet.Unlock()

	val := valSet.Validators[index]
	return val, nil
}

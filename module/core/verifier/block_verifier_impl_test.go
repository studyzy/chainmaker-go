/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBlockVerifierImpl_VerifyBlock(t *testing.T) {
	ctl := gomock.NewController(t)
	var chainId = "Chain1"

	msgBus := msgbus.NewMessageBus()
	txScheduler := mock.NewMockTxScheduler(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
	proposedCache := cache.NewProposalCache(mock.NewMockChainConf(ctl), ledgerCache)
	signerMember := mock.NewMockSigningMember(ctl)

	verifier := &BlockVerifierImpl{
		chainId:         chainId,
		msgBus:          msgBus,
		txScheduler:     txScheduler,
		snapshotManager: snapshotMgr,
		ledgerCache:     ledgerCache,
		blockchainStore: blockchainStoreImpl,
		reentrantLocks: &reentrantLocks{
			reentrantLocks: make(map[string]interface{}),
		},
		proposalCache: proposedCache,
		log:           logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
	}

	sig_default := []byte("DEFAULT_SIGNATURE")
	signerMember.EXPECT().Sign(gomock.Any(), gomock.Any()).Return(sig_default, nil).Times(100)

	b0 := cache.CreateNewTestBlock(0)
	ledgerCache.SetLastCommittedBlock(b0)
	b1 := cache.CreateNewTestBlock(1)
	require.Nil(t, verifier.VerifyBlock(b1, protocol.CONSENSUS_VERIFY))
}

func Test_ReentrantLock(t *testing.T) {
	lock := &reentrantLock{}

	for i := 0; i < 3; i++ {
		go func() {
			j := i
			if lock.lock("") {
				require.False(t, lock.lock(""))
				defer lock.unlock("")
				fmt.Println(fmt.Sprintf("%d get lock", j))
				time.Sleep(2 * time.Second)
			}
		}()
	}

	for i := 0; i < 3; i++ {
		j := i
		go func() {
			for {
				if lock.lock("") {
					defer lock.unlock("")
					fmt.Println(fmt.Sprintf("finally %d get lock", j))
					break
				}
			}
		}()
	}

	time.Sleep(5 * time.Second)
}

func Test_ReentrantLocks(t *testing.T) {
	locks := &reentrantLocks{
		reentrantLocks: make(map[string]interface{}),
	}
	for i := 0; i < 3; i++ {
		go func() {
			j := i
			if locks.lock("1") {
				require.False(t, locks.lock("1"))
				defer locks.unlock("1")
				fmt.Println(fmt.Sprintf("%d get lock", j))
				time.Sleep(2 * time.Second)
			}
		}()
	}

	for i := 0; i < 3; i++ {
		j := i
		go func() {
			for {
				if locks.lock("2") {
					defer locks.unlock("2")
					fmt.Println(fmt.Sprintf("finally %d get lock", j))
					time.Sleep(1 * time.Second)
					break
				}
			}
		}()
	}
	time.Sleep(5 * time.Second)

}

type reentrantLock struct {
	reentrantLock *int32
}

func (l *reentrantLock) lock(key string) bool {
	return atomic.CompareAndSwapInt32(l.reentrantLock, 0, 1)
}

func (l *reentrantLock) unlock(key string) bool {
	return atomic.CompareAndSwapInt32(l.reentrantLock, 1, 0)
}

func Test_Hashprefix(t *testing.T) {
	b := []byte(":B:1.0.0")
	require.True(t, strings.HasPrefix(string(b), protocol.ContractByteCode))
}

func Test_DispatchTask(t *testing.T) {
	tasks := utils.DispatchTxVerifyTask(nil)
	fmt.Println(tasks)
	txs := make([]*commonpb.Transaction, 0)
	for i := 0; i < 5; i++ {
		txs = append(txs, cache.CreateNewTestTx())
	}
	require.Equal(t, 5, len(txs))
	verifyTasks := utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
	}

	for i := 0; i < 123; i++ {
		txs = append(txs, cache.CreateNewTestTx())
	}
	verifyTasks = utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
	}

	for i := 0; i < 896; i++ {
		txs = append(txs, cache.CreateNewTestTx())
	}
	verifyTasks = utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
	}
}

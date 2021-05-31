/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker/common/random/uuid"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker-go/utils"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultRefreshTime    = time.Hour * 12        // refresh vmPool time, use for grow or shrink
	defaultMaxSize        = 100                   // the max pool size for every contract
	defaultMinSize        = 10                    // the min pool size
	defaultChangeSize     = 10                    // grow pool size
	defaultDelayTolerance = time.Millisecond * 10 // if get instance avg time greater than this value, should grow pool
	defaultApplyThreshold = 100                   // if apply times greater than this value, should grow pool
	defaultDiscardCount   = 10                    // if wasmer instance invoke error more than N times, should close and discard this instance
)

// VmPoolManager manages vm pools for all contracts
type VmPoolManager struct {
	// chain identifier
	chainId string
	// control map operations
	m sync.Mutex
	// contractName_contractVersion -> vm pool
	instanceMap map[string]*vmPool
	// module log
	log *logger.CMLogger
}

// vmPool, each contract has a vm pool providing multiple vm instances to call
// vm pool can grow and shrink on demand
type vmPool struct {
	// the corresponding contract info
	contractId *commonPb.ContractId
	byteCode   []byte
	// byteCode wasm module
	module *wasm.Module
	// wasm instance pool
	instances chan *wrappedInstance
	// current instance size in pool
	currentSize int32
	// use count from last refresh
	useCount int32
	// total delay (in ms) from last refresh
	totalDelay int32
	// total application count for pool grow
	// if we cannot get instance right now, applyGrowCount++
	applyGrowCount int32
	// apply signal channel
	applySignalC    chan struct{}
	closeC          chan struct{}
	resetC          chan struct{}
	removeInstanceC chan struct{}
	addInstanceC    chan struct{}
	log             *logger.CMLogger
}

// wrappedInstance wraps instance with id and other info
type wrappedInstance struct {
	// id
	id string
	// wasm instance provided by wasmer
	wasmInstance *wasm.Instance
	// lastUseTime, unix timestamp in ms
	lastUseTime int64
	// createTime, unix timestamp in ms
	createTime int64
	// errCount, current instance invoke method error count
	errCount int32
}

// NewVmPoolManager return VmPoolManager for every chain
func NewVmPoolManager(chainId string) *VmPoolManager {
	vmPoolManager := &VmPoolManager{
		instanceMap: make(map[string]*vmPool),
		log:         logger.GetLoggerByChain(logger.MODULE_VM, chainId),
		chainId:     chainId,
	}
	return vmPoolManager
}

// NewRuntimeInstance init vm pool and check byteCode correctness
func (m *VmPoolManager) NewRuntimeInstance(contractId *commonPb.ContractId, byteCode []byte) (*RuntimeInstance, error) {
	var err error
	if contractId == nil || contractId.ContractName == "" || contractId.ContractVersion == "" {
		err = fmt.Errorf("contract id is nil")
		m.log.Warn(err)
		return nil, err
	}

	if byteCode == nil || len(byteCode) == 0 {
		err = fmt.Errorf("[%s_%s], byte code is nil", contractId.ContractName, contractId.ContractVersion)
		m.log.Warn(err)
		return nil, err
	}

	pool, err := m.getVmPool(contractId, byteCode)
	if err != nil || pool == nil {
		return nil, err
	}

	runtime := &RuntimeInstance{
		pool:    pool,
		log:     m.log,
		chainId: m.chainId,
	}

	return runtime, nil
}

func (m *VmPoolManager) getVmPool(contractId *commonPb.ContractId, byteCode []byte) (*vmPool, error) {
	var err error
	key := contractId.ContractName + "_" + contractId.ContractVersion

	pool, ok := m.instanceMap[key]
	if !ok {
		m.m.Lock()
		defer m.m.Unlock()

		pool, ok = m.instanceMap[key]
		if !ok {
			start := utils.CurrentTimeMillisSeconds()
			m.log.Infof("[%s] init vm pool start", key)

			pool, err = newVmPool(contractId, byteCode, m.log)
			if err != nil {
				return nil, err
			}

			pool.grow(defaultMinSize)
			m.instanceMap[key] = pool
			end := utils.CurrentTimeMillisSeconds()
			m.log.Infof("[%s] init vmPool done, currentSize=%d, spend %dms", key, pool.currentSize, end-start)
		}
	}
	return pool, err
}

// GetInstance get a vm instance to run contract
// should be followed by defer resetInstance
func (p *vmPool) GetInstance() *wrappedInstance {
	atomic.AddInt32(&p.useCount, 1)

	// get instance from vm pool
	select {
	case instance := <-p.instances:
		// concurrency safe here
		instance.lastUseTime = utils.CurrentTimeMillisSeconds()
		return instance
	default:
		// nothing
	}

	// if we cannot get it right now, send apply signal and wait
	// add wait time to total delay
	curTimeMS1 := utils.CurrentTimeMillisSeconds()
	go func() {
		p.applySignalC <- struct{}{}
	}()

	instance := <-p.instances
	curTimeMS2 := utils.CurrentTimeMillisSeconds()
	instance.lastUseTime = curTimeMS2
	elapsedTimeMS := int32(curTimeMS2 - curTimeMS1)
	atomic.AddInt32(&p.totalDelay, elapsedTimeMS)

	return instance
}

// NewInstance create a wasmer instance directly, for cross contract call
func (p *vmPool) NewInstance() (*wrappedInstance, error) {
	return p.newInstanceFromModule()
}

// CloseInstance close a wasmer instance directly, for cross contract call
func (p *vmPool) CloseInstance(instance *wrappedInstance) {
	if instance != nil {
		CallDeallocate(instance.wasmInstance)
		instance.wasmInstance.Close()
		instance = nil
	}
}

// RevertInstance revert instance to pool
func (p *vmPool) RevertInstance(instance *wrappedInstance) {
	if p.shouldDiscard(instance) {
		go func() {
			p.removeInstanceC <- struct{}{}
			p.addInstanceC <- struct{}{}
		}()
	} else {
		p.instances <- instance
	}
}

func newVmPool(contractId *commonPb.ContractId, byteCode []byte, log *logger.CMLogger) (*vmPool, error) {
	if ok := wasm.Validate(byteCode); !ok {
		return nil, fmt.Errorf("[%s_%s], byte code validation failed", contractId.ContractName, contractId.ContractVersion)
	}

	module, err := wasm.Compile(byteCode)
	if err != nil {
		return nil, fmt.Errorf("[%s_%s], byte code compile failed", contractId.ContractName, contractId.ContractVersion)
	}

	vmPool := &vmPool{
		contractId:      contractId,
		byteCode:        byteCode,
		module:          &module,
		instances:       make(chan *wrappedInstance, defaultMaxSize),
		currentSize:     0,
		useCount:        0,
		totalDelay:      0,
		applyGrowCount:  0,
		applySignalC:    make(chan struct{}),
		removeInstanceC: make(chan struct{}),
		addInstanceC:    make(chan struct{}),
		closeC:          make(chan struct{}),
		resetC:          make(chan struct{}),
		log:             log,
	}

	instance, err := vmPool.newInstanceFromModule()
	if err != nil {
		return nil, fmt.Errorf("[%s_%s], byte code compile failed, %s", contractId.ContractName, contractId.ContractVersion, err.Error())
	}

	instance.wasmInstance.Close()
	log.Infof("vm pool verify byteCode finish.")

	go vmPool.startRefreshingLoop()
	log.Infof("vm pool startRefreshingLoop...")
	return vmPool, nil
}

// startRefreshingLoop refreshing loop manages the vm pool
// all grow and shrink operations are called here
func (p *vmPool) startRefreshingLoop() {

	refreshTimer := time.NewTimer(defaultRefreshTime)
	key := p.contractId.ContractName + "_" + p.contractId.ContractVersion
	for {
		select {
		case <-p.applySignalC:
			p.applyGrowCount++
			if p.shouldGrow() {
				p.grow(defaultChangeSize)
				p.applyGrowCount = 0
				p.log.Infof("[%s] vm pool grows by %d, the current size is %d", key, defaultChangeSize, p.currentSize)
			}
		case <-refreshTimer.C:
			p.log.Debugf("[%s] vm pool refresh timer expires. current size is %d, delay is %dms", key, p.currentSize, p.getAverageDelay())
			if p.shouldGrow() {
				p.grow(defaultChangeSize)
				p.applyGrowCount = 0
				p.log.Infof("[%s] vm pool grows by %d, the current size is %d", key, defaultChangeSize, p.currentSize)
			} else if p.shouldShrink() {
				p.shrink(defaultChangeSize)
				p.log.Infof("[%s] vm pool shrinks by %d, the current size is %d", key, defaultChangeSize, p.currentSize)
			}

			// other go routine may modify useCount & totalDelay
			// so we use atomic operation here
			atomic.StoreInt32(&p.useCount, 0)
			atomic.StoreInt32(&p.totalDelay, 0)
			refreshTimer.Reset(defaultRefreshTime)
		case <-p.closeC:
			refreshTimer.Stop()
			for p.currentSize > 0 {
				instance := <-p.instances
				CallDeallocate(instance.wasmInstance)
				instance.wasmInstance.Close()
				p.currentSize--
			}
			close(p.instances)
			return
		case <-p.resetC:
			for p.currentSize > 0 {
				instance := <-p.instances
				CallDeallocate(instance.wasmInstance)
				instance.wasmInstance.Close()
				p.currentSize--
			}
			close(p.instances)
			p.instances = make(chan *wrappedInstance, defaultMaxSize)
			p.grow(defaultMinSize)
		case <-p.removeInstanceC:
			p.currentSize--
		case <-p.addInstanceC:
			p.grow(1)
		}
	}
}

// shouldGrow grow vm pool when
// 1. current size + grow size <= max size, AND
// 2.1. apply count >= apply threshold, OR
// 2.2. average delay > delay tolerance (int operation here is safe)
func (p *vmPool) shouldGrow() bool {
	if p.currentSize < defaultMinSize {
		return true
	}
	if p.currentSize+defaultChangeSize <= defaultMaxSize {
		if p.applyGrowCount > defaultApplyThreshold {
			return true
		}

		if p.getAverageDelay() > int32(defaultDelayTolerance) {
			return true
		}

		if p.currentSize < int32(defaultMinSize) {
			return true
		}
	}
	return false
}

func (p *vmPool) grow(count int32) {
	for count > 0 {
		size := int32(10)
		if count < size {
			size = count
		}
		count -= size

		wg := sync.WaitGroup{}
		for i := int32(0); i < size; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				instance, _ := p.newInstanceFromModule()
				p.instances <- instance
				atomic.AddInt32(&p.currentSize, 1)
			}()
		}
		wg.Wait()
		p.log.Infof("vm pool grow size = %d", size)
	}
}

// shouldShrink shrink vm pool when
// 1. current size > min size, AND
// 2. average delay <= delay tolerance (int operation here is safe)
func (p *vmPool) shouldShrink() bool {
	if p.currentSize > defaultMinSize && p.getAverageDelay() <= int32(defaultDelayTolerance) && p.currentSize > defaultChangeSize {
		return true
	}
	return false
}

func (p *vmPool) shrink(count int32) {
	for i := int32(0); i < count; i++ {
		instance := <-p.instances
		CallDeallocate(instance.wasmInstance)
		instance.wasmInstance.Close()
		instance = nil
		p.currentSize--
	}
}

// shouldDiscard discard instance when
// error count times more than defaultDiscardCount
func (p *vmPool) shouldDiscard(instance *wrappedInstance) bool {
	if instance.errCount > defaultDiscardCount {
		return true
	}
	return false
}

func (p *vmPool) newInstanceFromByteCode() (*wrappedInstance, error) {
	vb := GetVmBridgeManager()
	wasmInstance, err := vb.NewWasmInstance(p.byteCode)
	if err != nil {
		p.log.Errorf("newInstanceFromByteCode fail: %s", err.Error())
		return nil, err
	}

	instance := &wrappedInstance{
		id:           uuid.GetUUID(),
		wasmInstance: &wasmInstance,
		lastUseTime:  utils.CurrentTimeMillisSeconds(),
		createTime:   utils.CurrentTimeMillisSeconds(),
		errCount:     0,
	}
	return instance, nil
}

func (p *vmPool) newInstanceFromModule() (*wrappedInstance, error) {
	vb := GetVmBridgeManager()
	wasmInstance, err := p.module.InstantiateWithImports(vb.GetImports())
	if err != nil {
		p.log.Errorf("newInstanceFromModule fail: %s", err.Error())
		return nil, err
	}

	instance := &wrappedInstance{
		id:           uuid.GetUUID(),
		wasmInstance: &wasmInstance,
		lastUseTime:  utils.CurrentTimeMillisSeconds(),
		createTime:   utils.CurrentTimeMillisSeconds(),
		errCount:     0,
	}
	return instance, nil
}

// getAverageDelay average delay calculation here maybe not so accurate due to concurrency
// but we can still use it to decide grow/shrink or not
func (p *vmPool) getAverageDelay() int32 {
	delay := atomic.LoadInt32(&p.totalDelay)
	count := atomic.LoadInt32(&p.useCount)
	if count == 0 {
		return 0
	}
	return delay / count
}

// reset the pool instances
func (p *vmPool) reset() {
	p.resetC <- struct{}{}
}

// close the pool
func (p *vmPool) close() {
	close(p.closeC)
}

// close the contract vm pool
func (m *VmPoolManager) closeAVmPool(contractId *commonPb.ContractId) {
	key := contractId.ContractName + "_" + contractId.ContractVersion
	pool, ok := m.instanceMap[key]
	if ok {
		m.log.Infof("close pool %s", key)
		pool.close()
	}
}

// close all contract vm pool
func (m *VmPoolManager) closeAllVmPool() {
	for key, pool := range m.instanceMap {
		m.log.Infof("close pool %s", key)
		pool.close()
	}
}

// FIXME: 确认函数名是否多了字符A？@taifu
// reset a contract vm pool install
func (m *VmPoolManager) resetAVmPool(contractId *commonPb.ContractId) {

	key := contractId.ContractName + "_" + contractId.ContractVersion
	pool, ok := m.instanceMap[key]
	if ok {
		m.log.Infof("reset pool %s", key)
		pool.reset()
	}
}

// reset all contract pool instance
func (m *VmPoolManager) ResetAllPool() {
	for key, pool := range m.instanceMap {
		m.log.Infof("reset pool %s", key)
		pool.reset()
	}
}

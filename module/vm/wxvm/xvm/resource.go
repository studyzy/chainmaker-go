/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package xvm

const (
	maxResourceLimit = 0xFFFFFFFF
)

// Limits describes the usage or limit of resources
type Limits struct {
	Cpu uint64
}

//// TotalGas converts resource to gas
//func (l *Limits) TotalGas(gasPrice *pb.GasPrice) int64 {
//	cpuGas := roundup(l.Cpu, gasPrice.GetCpuRate())
//	memGas := roundup(l.Memory, gasPrice.GetMemRate())
//	diskGas := roundup(l.Disk, gasPrice.GetDiskRate())
//	feeGas := roundup(l.XFee, gasPrice.GetXfeeRate())
//	return cpuGas + memGas + diskGas + feeGas
//}

// Add accumulates resource limits, returns self.
func (l *Limits) Add(l1 Limits) *Limits {
	l.Cpu += l1.Cpu
	return l
}

// Sub sub limits from l
func (l *Limits) Sub(l1 Limits) *Limits {
	l.Cpu -= l1.Cpu
	return l
}

// Exceed judge whether resource exceeds l1
func (l Limits) Exceed(l1 Limits) bool {
	return l.Cpu > l1.Cpu
}

// MaxLimits describes the maximum limit of resources
var MaxLimits = Limits{
	Cpu: maxResourceLimit,
}

//func roundup(n, scale int64) int64 {
//	if scale == 0 {
//		return 0
//	}
//	return (n + scale - 1) / scale
//}//unused(deadcode)

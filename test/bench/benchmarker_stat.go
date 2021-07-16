/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"sync/atomic"
	"time"

	gmetrics "github.com/rcrowley/go-metrics"
)

//BenchmarkStat is a stat to count by atomic
type BenchmarkStat struct {
	TxHeight     uint64
	TxSend       *int64
	TxTimeout    *int64
	TxSucc       *int64
	TxDuplicated *int64
	TxOtherFail  *int64
	TxTotalFail  *int64
	TpsMeter     gmetrics.Meter
}

func NewBenchmarkStat() *BenchmarkStat {
	bs := &BenchmarkStat{
		TxHeight:     0,
		TxSend:       new(int64),
		TxTimeout:    new(int64),
		TxSucc:       new(int64),
		TxDuplicated: new(int64),
		TxOtherFail:  new(int64),
		TxTotalFail:  new(int64),
		TpsMeter:     gmetrics.NewMeter(),
	}

	gmetrics.Register("benchmark_tps", bs.TpsMeter)
	return bs
}

func (bs *BenchmarkStat) StatInfo() string {
	s := fmt.Sprintf(`------------
Height:%d,
Send:%d,
Succ:%d,
Fail:%d, Timeout:%d, Duplicated:%d,
TPS real time:%f,
TPS moving average 1 min:%f,
Meter Count:%d,
------------
`,
		bs.TxHeight,
		atomic.LoadInt64(bs.TxSend),
		atomic.LoadInt64(bs.TxSucc),
		atomic.LoadInt64(bs.TxTotalFail),
		atomic.LoadInt64(bs.TxTimeout),
		atomic.LoadInt64(bs.TxDuplicated),
		bs.TpsMeter.RateMean(),
		bs.TpsMeter.Rate1(),
		bs.TpsMeter.Count(),
	)

	return s
}

func (bs *BenchmarkStat) PrintInfoLooper(inervalSec int) error {
	//looper print statistics
	ticker := time.NewTicker(time.Duration(inervalSec) * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println(bs.StatInfo())
		}
	}
}

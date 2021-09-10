/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"encoding/json"
	"time"
)

type roundMetrics struct {
	round                 int32
	enterNewRoundTime     time.Time
	enterProposalTime     time.Time
	enterPrevoteTime      time.Time
	enterPrecommitTime    time.Time
	enterCommitTime       time.Time
	persistStateDurations map[string][]time.Duration
}

func newRoundMetrics(round int32) *roundMetrics {
	return &roundMetrics{
		round:                 round,
		persistStateDurations: make(map[string][]time.Duration),
	}
}

type roundMetricsJson struct {
	Round                 int32
	Proposal              string
	Prevote               string
	Precommit             string
	PersistStateDurations map[string][]string
}

func (r *roundMetrics) roundMetricsJson() *roundMetricsJson {
	j := &roundMetricsJson{
		Round:                 r.round,
		Proposal:              r.enterPrevoteTime.Sub(r.enterProposalTime).String(),
		Prevote:               r.enterPrecommitTime.Sub(r.enterPrevoteTime).String(),
		Precommit:             r.enterCommitTime.Sub(r.enterPrecommitTime).String(),
		PersistStateDurations: make(map[string][]string),
	}

	for k, v := range r.persistStateDurations {
		for _, d := range v {
			j.PersistStateDurations[k] = append(j.PersistStateDurations[k], d.String())
		}
	}

	return j
}

func (r *roundMetrics) SetEnterNewRoundTime() {
	r.enterNewRoundTime = time.Now()
}

func (r *roundMetrics) SetEnterProposalTime() {
	r.enterProposalTime = time.Now()
}

func (r *roundMetrics) SetEnterPrevoteTime() {
	r.enterPrevoteTime = time.Now()
}

func (r *roundMetrics) SetEnterPrecommitTime() {
	r.enterPrecommitTime = time.Now()
}

func (r *roundMetrics) SetEnterCommitTime() {
	r.enterCommitTime = time.Now()
}

func (r *roundMetrics) AppendPersistStateDuration(step string, d time.Duration) {
	r.persistStateDurations[step] = append(r.persistStateDurations[step], d)
}

type heightMetrics struct {
	height             uint64
	enterNewHeightTime time.Time
	rounds             map[int32]*roundMetrics
}

func newHeightMetrics(height uint64) *heightMetrics {
	return &heightMetrics{
		height: height,
		rounds: make(map[int32]*roundMetrics),
	}
}

type heightMetricsJson struct {
	Height             uint64
	EnterNewHeightTime string
	Rounds             map[int32]*roundMetricsJson
}

func (h *heightMetrics) String() string {
	j := heightMetricsJson{
		Height:             h.height,
		EnterNewHeightTime: h.enterNewHeightTime.String(),
		Rounds:             map[int32]*roundMetricsJson{},
	}
	for k, v := range h.rounds {
		j.Rounds[k] = v.roundMetricsJson()
	}
	byt, _ := json.Marshal(j)
	return string(byt)
}

func (h *heightMetrics) roundString(num int32) string {
	j := heightMetricsJson{
		Height:             h.height,
		EnterNewHeightTime: h.enterNewHeightTime.String(),
		Rounds:             map[int32]*roundMetricsJson{},
	}
	j.Rounds[num] = h.rounds[num].roundMetricsJson()
	byt, _ := json.Marshal(j)
	return string(byt)
}

func (h *heightMetrics) SetEnterNewHeightTime() {
	h.enterNewHeightTime = time.Now()
}

func (h *heightMetrics) getRoundMertrics(round int32) *roundMetrics {
	if _, ok := h.rounds[round]; !ok {
		h.rounds[round] = newRoundMetrics(round)
	}

	return h.rounds[round]
}

func (h *heightMetrics) SetEnterNewRoundTime(round int32) {
	r := h.getRoundMertrics(round)
	r.SetEnterNewRoundTime()
}

func (h *heightMetrics) SetEnterProposalTime(round int32) {
	r := h.getRoundMertrics(round)
	r.SetEnterProposalTime()
}

func (h *heightMetrics) SetEnterPrevoteTime(round int32) {
	r := h.getRoundMertrics(round)
	r.SetEnterPrevoteTime()
}

func (h *heightMetrics) SetEnterPrecommitTime(round int32) {
	r := h.getRoundMertrics(round)
	r.SetEnterPrecommitTime()
}

func (h *heightMetrics) SetEnterCommitTime(round int32) {
	r := h.getRoundMertrics(round)
	r.SetEnterCommitTime()
}

func (h *heightMetrics) AppendPersistStateDuration(round int32, step string, d time.Duration) {
	r := h.getRoundMertrics(round)
	r.AppendPersistStateDuration(step, d)
}

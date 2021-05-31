/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package liveness

//func TestPacemaker_UpdateTC(t *testing.T) {
//	ts := timeservice.NewTimerService()
//	log := logger.GetLogger("test_chained_hotstuff")
//	pace := NewPacemaker(log, 0, 100, 1, ts)
//
//	// 1. Simulate the initialization of nodes
//	require.Truef(t, pace.ProcessCertificates(99, 100, 100, 96), "should enter next level")
//	require.EqualValues(t, 100, int(pace.GetHeight()))
//	require.EqualValues(t, 101, int(pace.GetCurrentLevel()))
//
//	// 2. enter next level with same height 99
//	require.Truef(t, pace.ProcessCertificates(99, 100, 101, 96), "should enter next level")
//	require.EqualValues(t, 100, int(pace.GetHeight()))
//	require.EqualValues(t, 102, int(pace.GetCurrentLevel()))
//
//	// 3. enter next height with same tc level, because may receive proposed block from another peer
//	// but not enter because maxLevel == currLevel
//	require.Falsef(t, pace.ProcessCertificates(100, 101, 101, 96), "should not enter next height")
//	require.EqualValues(t, 101, int(pace.GetHeight()))
//	require.EqualValues(t, 102, int(pace.GetCurrentLevel()))
//
//	// 4. enter next height with highest qc level
//	require.Truef(t, pace.ProcessCertificates(101, 102, 101, 96), "should enter next height")
//	require.EqualValues(t, 102, int(pace.GetHeight()))
//	require.EqualValues(t, 103, int(pace.GetCurrentLevel()))
//}

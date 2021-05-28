/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package protocol

type HotStuffHelper interface {
	// DiscardAboveHeight Delete blocks data greater than the baseHeight
	DiscardAboveHeight(baseHeight int64)
}

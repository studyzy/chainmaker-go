/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

//SyncServer is the server to sync the blockchain
type SyncService interface {
	//Init the sync server, and the sync server broadcast the current block height every broadcastTime
	Start() error

	//Stop the sync server
	Stop()
}

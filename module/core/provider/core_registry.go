/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	"reflect"
	"strings"
)

// nolint: deadcode, unused
type a interface {
}

var coreEngineRegistry = map[string]reflect.Type{}

// RegisterCoreEngineProvider add a type to the coreEngineRegistry. If type already registered, will panic.
func RegisterCoreEngineProvider(consensusType string, cp CoreProvider) {
	consensusType = strings.ToUpper(consensusType)
	_, found := coreEngineRegistry[consensusType]
	if found {
		panic("core engine provider[" + consensusType + "] already registered!")
	}
	coreEngineRegistry[consensusType] = reflect.TypeOf(cp)
}

// NewCoreEngineProviderByConsensusType create a new provider by name, returning it as CoreEngineProvider interface.
// If type not found, will panic.
func NewCoreEngineProviderByConsensusType(consensusType string) CoreProvider {
	consensusType = strings.ToUpper(consensusType)
	t, found := coreEngineRegistry[consensusType]
	if !found {
		panic("core engine provider[" + consensusType + "] not found!")
	}

	return reflect.New(t).Elem().Interface().(CoreProvider)
}

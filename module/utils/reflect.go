/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"reflect"
	"strconv"
)

// UpdateField set the key in data by params[key] to struct public field
// return if the field has been changed
// such as key->field: block_height -> BlockHeight
func UpdateField(params map[string][]byte, key string, config interface{}) (bool, error) {
	if valueB, ok := params[key]; ok {
		value := string(valueB)
		structElem := reflect.ValueOf(config).Elem()
		fieldName := ToCamelCase(key)
		field := structElem.FieldByName(fieldName)
		if !field.IsValid() {
			return false, fmt.Errorf("field[%s] not exist", fieldName)
		}
		switch field.Type().Name() {
		case "bool":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return false, err
			}
			sliceValue := reflect.ValueOf(b)
			field.Set(sliceValue)
		case "uint32":
			parseUint, err := strconv.ParseUint(value, 10, 0)
			if err != nil {
				return false, err
			}
			sliceValue := reflect.ValueOf(uint32(parseUint))
			field.Set(sliceValue)
		case "uint64":
			parseUint, err := strconv.ParseUint(value, 10, 0)
			if err != nil {
				return false, err
			}
			sliceValue := reflect.ValueOf(parseUint)
			field.Set(sliceValue)
		case "string":
			sliceValue := reflect.ValueOf(value)
			field.Set(sliceValue)
		default:
			return false, fmt.Errorf("no match type[%s], you should expand this util", field.Type().Name())
		}
		return true, nil
	}
	return false, nil
}

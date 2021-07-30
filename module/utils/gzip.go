/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// GZipCompressBytes compress bytes with GZip(BestSpeed mode).
func GZipCompressBytes(data []byte) ([]byte, error) {
	var input bytes.Buffer
	g, err := gzip.NewWriterLevel(&input, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	_, err = g.Write(data)
	if err != nil {
		return nil, err
	}
	err = g.Close()
	if err != nil {
		return nil, err
	}
	return input.Bytes(), nil
}

// GZipDeCompressBytes decompress bytes with GZip.
func GZipDeCompressBytes(data []byte) ([]byte, error) {
	var in bytes.Buffer
	in.Write(data)
	r, err := gzip.NewReader(&in)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

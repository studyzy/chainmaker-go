/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"compress/gzip"
	"io"
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
	var out bytes.Buffer
	var in bytes.Buffer
	in.Write(data)
	r, err := gzip.NewReader(&in)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, err
	}
	err = r.Close()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

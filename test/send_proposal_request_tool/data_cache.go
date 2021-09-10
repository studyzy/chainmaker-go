/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"
	"sync"

	"chainmaker.org/chainmaker/utils/v2"
)

//FileCacheReader read file data and cache to memory,when invoke read method, if file has read,get from memory cache
type FileCacheReader struct {
	cache sync.Map
}

func NewFileCacheReader() FileCacheReader {
	return FileCacheReader{}
}

//Read get file data from cache
func (fc *FileCacheReader) Read(filePath string) *[]byte {
	fileData, ok := fc.cache.Load(filePath)
	if ok {
		return fileData.(*[]byte)
	}
	data, err := ioutil.ReadFile(filePath)
	if nil != err {
		panic(err)
	}
	fc.cache.Store(filePath, &data)
	return &data
}

//CertFileCache cache cert data
type CertFileCacheReader struct {
	cache sync.Map
}

func NewCertFileCacheReader() CertFileCacheReader {
	return CertFileCacheReader{}
}

//Read get cert data from cache
func (cfc *CertFileCacheReader) Read(filePath string, data []byte, hashType string) (*[]byte, error) {
	fileData, ok := cfc.cache.Load(filePath)
	if ok {
		return fileData.(*[]byte), nil
	}
	certificateId, err := utils.GetCertificateId(data, hashAlgo)
	if err != nil {
		return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
	}
	cfc.cache.Store(filePath, &certificateId)
	return &certificateId, nil
}

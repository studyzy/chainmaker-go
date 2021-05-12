/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {

	fileCount := 1
	files := createFile(fileCount)
	defer removeFile(files)

	fileCacheReader := NewFileCacheReader()
	fileData := fileCacheReader.Read(files[0])

	assert.NotNil(t, fileData, "")
	assert.NotEqual(t, len(*fileData), "")
}

func TestCacheParallel(t *testing.T) {
	fileCount := 10
	runTimes := 20000

	files := createFile(fileCount)

	defer removeFile(files)

	reader := NewFileCacheReader()

	group := sync.WaitGroup{}

	group.Add(1)
	go func() {
		i := 0
		for j := 0; j < runTimes; j++ {
			data := reader.Read(files[i])
			assert.NotNil(t, data)
			i++
			if i >= fileCount {
				i = 0
			}
		}
		group.Done()
	}()

	group.Add(1)
	go func() {
		i := 0
		for j := 0; j < runTimes; j++ {
			data := reader.Read(files[i])
			assert.NotNil(t, data)
			i++
			if i >= fileCount {
				i = 0
			}
		}
		group.Done()
	}()

	group.Wait()
}

//create file, and return file paths
func createFile(count int) []string {
	temp := "test"
	result := make([]string, count)
	for i := 0; i < count; i++ {
		file, err := ioutil.TempFile("", "data_cache_test")
		if err != nil {
			panic(err)
		}
		_, err = file.WriteString(temp)
		if err != nil {
			panic(err)
		}
		err = file.Close()
		if err != nil {
			panic(err)
		}
		result[i] = file.Name()
	}

	return result
}

func removeFile(files []string) {
	for _, v := range files {
		err := os.Remove(v)
		if err != nil {
			panic(err)
		}

	}
}

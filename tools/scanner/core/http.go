/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	contentType = "application/json"
)

type resultModel struct {
	Code    int    `json:"retcode"`
	Message string `json:"retmsg"`
}

func post(url string, data interface{}) (*resultModel, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	jsonString, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal mail error: %s", err)
	}
	resp, err := client.Post(url, contentType, bytes.NewBuffer(jsonString))
	if err != nil {
		return nil, fmt.Errorf("http post error: %s", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error: %s", err)
	}

	result := &resultModel{}
	if err = json.Unmarshal(respBody, result); err != nil {
		return nil, fmt.Errorf("unmarshal result error: %s", err)
	}
	return result, nil
}

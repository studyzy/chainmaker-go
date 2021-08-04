/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/scanner/config"
)

type mailModel struct {
	// 告警邮件标题
	Subjuct string `json:"subject"`
	// 告警邮件内容
	Content string `json:"content"`
	// 告警邮件接收人
	// - UserID1;UserID2;UserID3： 指定接收消息的成员，成员ID列表，多个使用 ";" 进行分割
	Receiver string `json:"receiver" `
	// 附件，base64编码内容，大小限制在配置文件attach_file_size指定
	AttachFile string `json:"attach_file"`
	// 附件名
	AttachName string `json:"attach_name"`
}

func SendMail(subject, content, receiver, attachFile, attachName string) error {
	body := &mailModel{
		subject,
		content,
		receiver,
		attachFile,
		attachName,
	}
	result, err := post(config.GlobalConfig.AlarmCenterConfig.SendMailURL, body)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("send mail error: %s", result.Message)
	}
	return nil
}

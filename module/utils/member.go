/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import "chainmaker.org/chainmaker/pb-go/accesscontrol"

type MemberGetter interface {
	//GetFullMemberInfo根据CERT_HASH获得完整的Cert，根据DID获得DID Document
	GetFullMemberInfo(memberId []byte,mtype accesscontrol.MemberType) ([]byte,error)
}
func GetMemberPubKeySA(member *accesscontrol.SerializedMember,getter MemberGetter) ([]byte,uint32){
	return []byte("pubkey"),member.SignatureAlgorithm
}
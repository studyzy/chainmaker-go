/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package statesqldb

import (
	"crypto/sha256"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
)

// MemberExtraInfo defines mysql orm model, used to create mysql table 'member_extra_infos'
type MemberExtraInfo struct {
	MemberHash []byte `gorm:"size:32;primaryKey"`
	MemberType int    `gorm:""`
	MemberInfo []byte `gorm:"size:2000"`
	OrgId      string `gorm:"size:200"`
	Seq        uint64 //防止Sequence是数据库关键字
}

func (b *MemberExtraInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.MemberHash, &b.MemberType, &b.MemberInfo, &b.OrgId, &b.Seq)
}
func (b *MemberExtraInfo) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE member_extra_infos (
    member_hash binary(32) primary key,
    member_type int,
	member_info blob(2000),
    org_id varchar(200),
    seq bigint default 0
    ) default character set utf8`
	} else if dbType == localconf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE member_extra_infos (
	member_hash blob primary key,
    member_type integer,
	member_info blob,
    org_id text,
    seq integer
    )`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *MemberExtraInfo) GetTableName() string {
	return "member_extra_infos"
}
func (b *MemberExtraInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO member_extra_infos values(?,?,?,?,?)",
		[]interface{}{b.MemberHash, b.MemberType, b.MemberInfo, b.OrgId, b.Seq}
}
func (b *MemberExtraInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE member_extra_infos set member_type=?,member_info=?,org_id=?,seq=?" +
			" WHERE member_hash=?",
		[]interface{}{b.MemberType, b.MemberInfo, b.OrgId, b.Seq, b.MemberHash}
}
func (b *MemberExtraInfo) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM member_extra_infos WHERE member_hash=?",
		[]interface{}{b.MemberHash}
}
func (b *MemberExtraInfo) GetSaveSql() (string, []interface{}) {
	if b.Seq > 1 { //update
		return "UPDATE member_extra_infos set seq=? WHERE member_hash=?",
			[]interface{}{b.Seq, b.MemberHash}
	}
	return b.GetInsertSql()
}

func NewMemberExtraInfo(member *accesscontrol.Member, extra *accesscontrol.MemberExtraData) *MemberExtraInfo {

	hash := getMemberHash(member)
	return &MemberExtraInfo{
		MemberHash: hash,
		MemberType: int(member.MemberType),
		MemberInfo: member.MemberInfo,
		OrgId:      member.OrgId,
		Seq:        extra.Sequence,
	}
}
func getMemberHash(member *accesscontrol.Member) []byte {
	data, _ := member.Marshal()
	hash := sha256.Sum256(data)
	return hash[:]
}
func (b *MemberExtraInfo) GetExtraData() *accesscontrol.MemberExtraData {
	return &accesscontrol.MemberExtraData{Sequence: b.Seq}
}

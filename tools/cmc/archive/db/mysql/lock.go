// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mysql

import (
	"database/sql"
	"log"
	"time"

	"gorm.io/gorm"
)

const DefaultLockLeaseAge = 10 * time.Second

type lock struct {
	ID        int64        `gorm:"primaryKey;type:int unsigned NOT NULL AUTO_INCREMENT"`
	CreatedAt time.Time    `gorm:"type:timestamp;not null"`
	UpdatedAt sql.NullTime `gorm:"type:timestamp"`
	ExpiredAt time.Time    `gorm:"type:timestamp;not null"`
	Holder    string       `gorm:"unique;not null"`
}

type dbLocker struct {
	db       *gorm.DB
	stopCh   chan struct{}
	holder   string
	leaseAge time.Duration
}

func NewDbLocker(db *gorm.DB, holder string, lease time.Duration) *dbLocker {
	return &dbLocker{
		db:       db,
		stopCh:   make(chan struct{}),
		holder:   holder,
		leaseAge: lease,
	}
}

func (locker *dbLocker) Lock() {
	for {
		err := locker.cleanExpired()
		if err != nil {
			log.Printf("%s\ntry lock db, wait %f seconds ...", err, DefaultLockLeaseAge.Seconds())
			time.Sleep(DefaultLockLeaseAge)
			continue
		}

		now := time.Now()
		err = locker.db.Create(&lock{
			CreatedAt: now,
			ExpiredAt: now.Add(locker.leaseAge),
			Holder:    locker.holder,
		}).Error
		if err != nil {
			log.Printf("%s\ntry lock db, wait %f seconds ...", err, DefaultLockLeaseAge.Seconds())
			time.Sleep(DefaultLockLeaseAge)
			continue
		}
		break
	}

	locker.startLease()
}

func (locker *dbLocker) UnLock() {
	locker.stopLease()
	locker.db.Where("holder = ?", locker.holder).Delete(&lock{})
}

func (locker *dbLocker) cleanExpired() error {
	return locker.db.Where("expired_at < ?", time.Now()).Delete(&lock{}).Error
}

func (locker *dbLocker) startLease() {
	go func() {
		// Refresh the lease when time elapses 3/4 of the locker.leaseAge
		ticker := time.NewTicker(locker.leaseAge * 3 / 4)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := locker.refreshLease()
				if err != nil {
					log.Printf("refreash lease err: %s\n", err)
				}
			case <-locker.stopCh:
				return
			}
		}
	}()
}

func (locker *dbLocker) stopLease() {
	close(locker.stopCh)
}

func (locker *dbLocker) refreshLease() error {
	return locker.db.Model(&lock{}).Where("holder = ?", locker.holder).
		Update("expired_at", time.Now().Add(locker.leaseAge)).Error
}

// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
// Datastore for perstiance of access rules, clients & keys
package atm

import (
	"database/sql"
	"errors"
)

type Datastore struct {
	pool *sql.DB
}

func NewDatastore(driver, dsn string) (*Datastore, error) {
	ds := &Datastore{}
	var err error
	ds.pool, err = sql.Open(driver, dsn)
	if nil != err {
		return nil, err
	}
	err = ds.Ping()
	if nil != err {
		return nil, err
	}
	return ds, nil
}

func (d *Datastore) Ping() error {
	return d.pool.Ping()
}

func (d *Datastore) Close() error {
	return d.pool.Close()
}

func (d *Datastore) KeyForRequest(u *UrlRequest, appId string) (string, int64, error) {
	var signing_key string
	var duration int64
	stmt, err := d.pool.Prepare("SELECT acc.signing_key, r.duration as duration from applications app, accounts acc, rules r " +
		"WHERE r.application_id = app.id AND r.account_id=acc.id AND app.id = ? AND acc.name = ? AND " +
		"? REGEXP r.container AND ? REGEXP r.object AND r.method = ?")
	if nil != err {
		return signing_key, duration, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(appId, u.Account, u.Container, u.Object, u.Method)
	if nil != err {
		return signing_key, duration, err
	}
	defer rows.Close()
	numRows := 0
	for rows.Next() {
		if numRows > 1 {
			return signing_key, duration, errors.New("Too many results")
		}
		err := rows.Scan(&signing_key, &duration)
		if nil != err {
			return signing_key, duration, err
		}
		err = rows.Err()
		if nil != err {
			return signing_key, duration, err
		}
		numRows++
	}

	return signing_key, duration, nil
}

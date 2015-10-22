// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
// Datastore for perstiance of access rules, clients & keys
package atm

import (
	"database/sql"
	"errors"
	"fmt"
)

type Datastore struct {
	pool        *sql.DB
	signingKeys Cache
}

type Account struct {
	Id   string `json:id`
	Name string `json:name`
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
	ds.signingKeys = NewCache()
	return ds, nil
}

func (d *Datastore) Ping() error {
	return d.pool.Ping()
}

func (d *Datastore) Close() error {
	return d.pool.Close()
}

func (d *Datastore) AddSigningKeyForAccount(key, account string) {
	d.signingKeys.Set(account, key)
}

func (d *Datastore) signingKeyFor(account string) string {
	return d.signingKeys.Get(account)
}

func (d *Datastore) Account(name string) (*Account, error) {
	a := &Account{}
	stmt, err := d.pool.Prepare("SELECT id, name from accounts where name = ?")
	if nil != err {
		return a, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(name)
	if nil != err {
		return a, err
	}
	defer rows.Close()
	numRows := 0
	for rows.Next() {
		if numRows > 1 {
			return a, errors.New("Too many results")
		}
		err := rows.Scan(&a.Id, &a.Name)
		if nil != err {
			return a, err
		}
		err = rows.Err()
		if nil != err {
			return a, err
		}
		numRows++
	}
	return a, nil
}

func (d *Datastore) KeyForRequest(u *UrlRequest, appId string) (string, int64, error) {
	var signing_key string
	var duration int64
	stmt, err := d.pool.Prepare("SELECT a.id, r.duration as duration from accounts a, rules r " +
		"WHERE r.account_id=a.id AND requestor_id = ? AND a.name = ? AND " +
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
	var grantingAccountId string
	for rows.Next() {
		if numRows > 1 {
			return signing_key, duration, errors.New("Too many results")
		}
		err := rows.Scan(&grantingAccountId, &duration)
		if nil != err {
			return signing_key, duration, err
		}
		err = rows.Err()
		if nil != err {
			return signing_key, duration, err
		}
		numRows++
	}
	if 0 == numRows {
		return "", 0, nil
	}

	signing_key = d.signingKeyFor(grantingAccountId)
	if "" == signing_key {
		return signing_key, 0, errors.New(fmt.Sprintf("Key not set for %s", u.Account))
	}

	return signing_key, duration, nil
}

func (d *Datastore) ApiKeySecret(apiKey string) (string, error) {
	var secret string
	stmt, err := d.pool.Prepare("SELECT secret from accounts where id = ?")
	if nil != err {
		return secret, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(apiKey)
	if nil != err {
		return secret, err
	}
	defer rows.Close()
	numRows := 0
	for rows.Next() {
		if numRows > 1 {
			return secret, errors.New("Too many results")
		}
		err := rows.Scan(&secret)
		if nil != err {
			return "", err
		}
		err = rows.Err()
		if nil != err {
			return "", err
		}
		numRows++
	}
	if 0 == numRows || "" == secret {
		return "", errors.New(fmt.Sprintf("No secret key for api: %s", apiKey))
	}

	return secret, nil
}

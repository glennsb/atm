// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
// Datastore for perstiance of access rules, clients & keys
package atm

import (
	"database/sql"
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

func (d *Datastore) Authorized(u *UrlRequest, appId string) bool {
	return false
}

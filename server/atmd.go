// ATM - Automatic TempUrl Maker
// A builder of Swift TempURLs
// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/user"

	_ "github.com/go-sql-driver/mysql"

	"github.com/glennsb/atm"
	"github.com/howeyc/gopass"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
)

const (
	HOST     = "https://o3.omrf.org"
	DURATION = int64(300)
)

var (
	Key              string
	Database         string
	Database_host    string
	Database_user    string
	Database_pass    string
	Database_port    int
	Default_duration int64
	Object_host      string
	ds               *atm.Datastore
)

func init() {
	current_user, _ := user.Current()
	Database_user = current_user.Username
	parseFlags()

	fmt.Printf("%s@%s/%s password: ", Database_user, Database_host, Database)
	Database_pass = string(gopass.GetPasswd())
}

func parseFlags() {
	flag.StringVar(&Database, "database", "atm", "Database name")
	flag.StringVar(&Database_host, "database-host", "localhost", "Database server hostname")
	flag.IntVar(&Database_port, "database-port", 3306, "Database server port")
	flag.StringVar(&Database_user, "database-user", Database_user, "Username for database")
	flag.Int64Var(&Default_duration, "duration", DURATION, "Default lifetime of tempurl")
	flag.StringVar(&Object_host, "host", HOST, "Swift host prefix")

	flag.Parse()
}

func keyFinder(a string) (string, error) {
	return ds.ApiKeySecret(a)
}

func main() {
	var err error
	ds, err = atm.NewDatastore("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		Database_user, Database_pass, Database_host,
		Database_port, Database))
	if nil != err {
		log.Fatal(err)
	}
	Database_pass = ""
	defer ds.Close()

	e := echo.New()

	// Middleware
	e.Use(mw.Logger())
	e.Use(mw.Recover())
	auth_opts := atm.NewHmacOpts(keyFinder)
	e.Use(atm.HMACAuth(auth_opts))

	e.Post("/urls", createUrl)
	e.Put("/keys/:name", setKey)
	e.Run(":8080")
}

type keyRequest struct {
	Key string `json:key`
}

func setKey(c *echo.Context) error {
	k := &keyRequest{}
	if err := c.Bind(k); nil != err {
		return c.JSON(http.StatusBadRequest, atm.ErrMsg(err.Error()))
	}
	a, err := ds.Account(c.Param("name"))
	if nil != err || a.Id == "" {
		return c.JSON(http.StatusGone, atm.ErrMsg(http.StatusText(http.StatusNotFound)))
	}
	if c.Get(atm.API_KEY) != a.Id {
		return c.JSON(http.StatusForbidden, atm.ErrMsg("Not authorized for this account"))
	}
	ds.AddSigningKeyForAccount(k.Key, a.Id)
	return c.JSON(http.StatusOK, a)
}

func createUrl(c *echo.Context) error {
	o := &atm.UrlRequest{Host: Object_host, Duration: Default_duration}
	if err := c.Bind(o); nil != err {
		return c.JSON(http.StatusBadRequest, atm.ErrMsg(err.Error()))
	}

	if !o.Valid() {
		return c.JSON(http.StatusBadRequest, atm.ErrMsg("Missing account, container, object, or method"))
	}

	duration := int64(0)
	var err error
	requestorId, ok := c.Get(atm.API_KEY).(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, atm.ErrMsg("Failed getting requesting id"))
	}
	o.Key, duration, err = ds.KeyForRequest(o, requestorId)
	if nil != err {
		log.Printf("keyForRequest: %v, %s. Error: %s", o, "", err.Error())
		return c.JSON(http.StatusInternalServerError, atm.ErrMsg("Trouble checking authorization"))
	}
	if "" == o.Key {
		return c.JSON(http.StatusForbidden, atm.ErrMsg("Not authorized for this resource"))
	}
	if duration > 0 && duration > o.Duration {
		o.Duration = duration
	}

	u := &atm.Tmpurl{
		Url:  o.SignedUrl(),
		Path: o.Path(),
	}

	c.Response().Header().Set("Location", u.Url)
	return c.JSON(http.StatusCreated, u)
}
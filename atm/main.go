// ATM - Automatic TempUrl Maker
// A builder of Swift TempURLs
// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
package main

import (
	"net/http"
	"os"

	//_ "github.com/go-sql-driver/mysql"

	"github.com/glennsb/atm"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
)

const (
	HOST     = "https://o3.omrf.org"
	DURATION = int64(300)
)

var Key string

func main() {
	Key = os.Args[1]

	e := echo.New()

	// Middleware
	e.Use(mw.Logger())
	e.Use(mw.Recover())

	e.Post("/urls", createUrl)
	e.Run(":8080")
}

func createUrl(c *echo.Context) error {
	o := &atm.UrlRequest{Host: HOST, Key: Key, Duration: DURATION}
	if err := c.Bind(o); nil != err {
		return c.JSON(http.StatusBadRequest, atm.ErrMsg(err.Error()))
	}

	if !o.Valid() {
		return c.JSON(http.StatusBadRequest, atm.ErrMsg("Missing account, container, object, or method"))
	}
	u := &atm.Tmpurl{
		Url:  o.SignedUrl(),
		Path: o.Path(),
	}

	c.Response().Header().Set("Location", u.Url)
	return c.JSON(http.StatusCreated, u)
}

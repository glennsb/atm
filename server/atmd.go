// ATM - Automatic TempUrl Maker
// A builder of Swift TempURLs
// Copyright (c) 2015 Stuart Glenn
// All rights reserved
// Use of this source code is goverened by a BSD 3-clause license,
// see included LICENSE file for details
package main

import (
	"fmt"
	"log"
	"os/user"

	"github.com/codegangsta/cli"
	"github.com/glennsb/atm"
	"github.com/howeyc/gopass"
)

func main() {
	app := cli.NewApp()
	app.Name = "atm"
	app.Usage = "Automated TempURL Maker"
	app.Version = "0.0.1 - 20151025"
	app.Author = "Stuart Glenn"
	app.Email = "Stuart-Glenn@omrf.org"
	app.Copyright = "2015 Stuart Glenn, All rights reserved"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "V, verbose",
			Usage: "show more output",
		},
	}

	app.Commands = []cli.Command{
		serverCommand(),
	}

	app.RunAndExitOnError()
}

func serverFlags() []cli.Flag {
	current_user, err := user.Current()
	default_username := ""
	if nil == err {
		default_username = current_user.Username
	}
	return []cli.Flag{
		cli.StringFlag{
			Name:  "database",
			Usage: "name of database",
			Value: "atm",
		},
		cli.StringFlag{
			Name:  "database-host",
			Usage: "hostname of database server",
			Value: "localhost",
		},
		cli.StringFlag{
			Name:  "database-user",
			Usage: "username for database connection",
			Value: default_username,
		},
		cli.IntFlag{
			Name:  "database-port",
			Usage: "port number of database server",
			Value: 3306,
		},
		cli.DurationFlag{
			Name:  "duration",
			Usage: "Default lifetime for generated tempurl",
			Value: atm.DURATION,
		},
		cli.StringFlag{
			Name:  "object-host, host",
			Usage: "Swift service host prefix",
			Value: atm.HOST,
		},
	}
}

func serverCommand() cli.Command {
	return cli.Command{
		Name:  "server",
		Usage: "Run webservice",
		Flags: serverFlags(),
		Action: func(c *cli.Context) {
			db_user := c.String("database-user")
			db_host := c.String("database-host")
			db := c.String("database")

			fmt.Printf("%s@%s/%s password: ", db_user, db_host, db)
			db_pass := string(gopass.GetPasswd())

			ds, err := atm.NewDatastore("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				db_user, db_pass, db_host, c.Int("database-port"), db))
			if nil != err {
				log.Fatal(err)
			} else {
				db_pass = ""
				defer ds.Close()

				service := &atm.Server{
					Ds:               ds,
					Object_host:      c.String("object-host"),
					Default_duration: int64(c.Duration("duration").Minutes()),
				}
				service.Run()
			}
		},
	}
}

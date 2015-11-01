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
	"os"
	"os/user"

	"github.com/codegangsta/cli"
	"github.com/glennsb/atm"
	"github.com/howeyc/gopass"
)

func main() {
	app := cli.NewApp()
	app.Name = "atm"
	app.Usage = "Automated TempURL Maker"
	app.Version = "0.0.2 - 20151026"
	app.Author = "Stuart Glenn"
	app.Email = "Stuart-Glenn@omrf.org"
	app.Copyright = "2015 Stuart Glenn, All rights reserved"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "V, verbose",
			Usage: "show more output",
		},
	}

	app.Commands = clientCommands()
	app.Commands = append(app.Commands, serverCommand())
	app.RunAndExitOnError()
}

func clientCommands() []cli.Command {
	return []cli.Command{
		cli.Command{
			Name:        "url",
			Usage:       "Request a temp url to Account/Container/Object",
			ArgsUsage:   "<Account> <Container> <Object> ",
			Description: "Send a request to the ATM service for a tempurl",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "api-key, k",
					Usage:  "account/user atm api-key",
					EnvVar: "ATM_API_KEY",
				},
				cli.StringFlag{
					Name:   "api-secret, s",
					Usage:  "account/user atm api-secret",
					EnvVar: "ATM_API_SECRET",
				},
				cli.StringFlag{
					Name:   "atm-host, a",
					Usage:  "atm server endpoint",
					EnvVar: "ATM_HOST",
				},
				cli.StringFlag{
					Name:  "method, m",
					Usage: "HTTP method requested for temp url",
					Value: "GET",
				},
			},
			Action: func(c *cli.Context) {
				method := c.String("method")
				if "" == method {
					fmt.Fprintf(os.Stderr, "Missing HTTP method option\n")
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				account := c.Args().Get(0)
				if "" == account {
					fmt.Fprintf(os.Stderr, "Missing Account argument\n")
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				container := c.Args().Get(1)
				if "" == container {
					fmt.Fprintf(os.Stderr, "Missing Container argument\n")
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				object := c.Args().Get(2)
				if "" == object {
					fmt.Fprintf(os.Stderr, "Missing Object argument\n")
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				atm := &atm.AtmClient{
					ApiKey:    c.String("api-key"),
					ApiSecret: c.String("api-secret"),
					AtmHost:   c.String("atm-host"),
				}
				url, err := atm.RequestTempUrl(method, account, container, object)
				if nil != err {
					log.Fatal(err)
					return
				}
				fmt.Println(url)
			},
		},

		cli.Command{
			Name:  "key",
			Usage: "Add/Remove signing key",
			Action: func(c *cli.Context) {
				log.Fatal("Not implemented yet")
				return
			},
		},
	}
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
				return
			}
			db_pass = ""
			defer ds.Close()

			service := &atm.Server{
				Ds:               ds,
				Object_host:      c.String("object-host"),
				Default_duration: int64(c.Duration("duration").Seconds()),
				Nonces:           atm.NewNonceStore(),
			}
			service.Run()
		},
	}
}

package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/zaru/jobcan/account"
	"github.com/zaru/jobcan/config"
)

func main() {
	app := cli.NewApp()
	app.Name = "jobcan"
	app.Usage = "attendance operation command for jobcan"
	app.Version = "0.2.4"
	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "jobcan init / initialize to jobcan account",
			Action: func(c *cli.Context) error {
				config.Init()
				return nil
			},
		},
		{
			Name:  "start",
			Usage: "jobcan start / I will start a job.",
			Action: func(c *cli.Context) error {
				a := account.New(account.Admin)
				a.Login()
				a.ExecAttendance("work_start")
				return nil
			},
		},
		{
			Name:  "end",
			Usage: "jobcan end / Today's work is over!",
			Action: func(c *cli.Context) error {
				a := account.New(account.Admin)
				a.Login()
				a.ExecAttendance("work_end")
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "jobcan list / Get your attendance list",
			Action: func(c *cli.Context) error {
				a := account.New(account.Admin)
				a.Login()
				err := a.ExecGetAttendance()
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
		{
			Name:  "show",
			Usage: "jobcan show [YYYYMMDD] / Show and fix time work for the specified day.",
			Action: func(c *cli.Context) error {
				a := account.New(account.Admin)
				a.Login()
				err := a.ExecGetAttendanceByDay(c.Args().First())
				if err != nil {
					return cli.NewExitError(err, 1)
				}
				return nil
			},
		},
	}

	app.Run(os.Args)

}

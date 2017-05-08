package main

import (
	"github.com/jgautheron/slashecs/bot"
	"github.com/urfave/cli"
)

// GosearchCommand handles all gosearch-related actions.
func StartCommand() cli.Command {
	return cli.Command{
		Name:  "start",
		Usage: "Start the slack bot",
		Action: func(c *cli.Context) error {
			bot.New().Init()
			return nil
		},
	}
}

package bot

import (
	"regexp"

	"github.com/jgautheron/slashecs/deployer"
	"github.com/nlopes/slack"
)

type ProcessFn func(*slack.RTM, *slack.MessageEvent, []string) error

var (
	AvailableCommands = map[*regexp.Regexp]ProcessFn{
		regexp.MustCompile(`^deploy ([\w\d\-\_]+) ([\w\d\-\_]+)$`): deployer.Deploy,
	}
)

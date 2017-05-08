package bot

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	. "github.com/jgautheron/slashecs/config"
	"github.com/nlopes/slack"
)

var (
	logger  = log.WithField("prefix", "bot")
	signals = make(chan os.Signal, 1)
)

type Bot struct {
	rtm *slack.RTM
}

func New() *Bot {
	api := slack.New(Config.SlackToken)
	return &Bot{
		rtm: api.NewRTM(),
	}
}

func (b *Bot) Init() {
	go b.rtm.ManageConnection()
	go b.monitorSlackEvents()

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-signals:
		logger.Warn("Termination signal caught, terminating slashecs...")
		close(signals)
	}
}

func (b *Bot) monitorSlackEvents() {
Loop:
	for {
		select {
		case msg := <-b.rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				logger.Debug("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				info := b.rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)
				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					go b.processMessage(ev, prefix)
				}

			case *slack.RTMError:
				logger.Errorf("Error while connecting to Slack: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				logger.Error("Invalid credentials")
				break Loop
			}
		}
	}
}

func (b *Bot) processMessage(msg *slack.MessageEvent, prefix string) {
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	for r, cmd := range AvailableCommands {
		matches := r.FindStringSubmatch(text)
		if len(matches) > 0 {
			logger.WithField("regex", r).Debug("The command matched")
			go cmd(b.rtm, msg, matches)
			// For now no chaining possibilities
			break
		}
	}
}

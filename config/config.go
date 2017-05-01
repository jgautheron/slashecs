package config

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
)

var Config cfg

type cfg struct {
	GithubAccessToken  string `envconfig:"GITHUB_ACCESS_TOKEN"`
	AwsRegion          string `envconfig:"AWS_REGION" default:"eu-west-1"`
	AwsAccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	AwsSecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
	SlackToken         string `envconfig:"SLACK_TOKEN" required:"true"`
	LogLevel           string `envconfig:"LOG_LEVEL" default:"info"`
	Cluster            string `envconfig:"CLUSTER" required:"true"`
}

func InitializeConfig() {
	if err := envconfig.Process("", &Config); err != nil {
		log.Fatal(err)
	}
}

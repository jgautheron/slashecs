package deployer

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	. "github.com/jgautheron/slashecs/config"
	"github.com/nlopes/slack"
)

const (
	TimeLimit = 3600 * time.Second
)

var (
	logger = log.WithField("prefix", "deployer")
)

func Deploy(rtm *slack.RTM, msg *slack.MessageEvent, matches []string) error {
	service, tag := matches[1], matches[2]

	sess := session.New(&aws.Config{
		Region: aws.String(Config.AwsRegion),
		Credentials: credentials.NewStaticCredentials(
			Config.AwsAccessKeyID,
			Config.AwsSecretAccessKey,
			"",
		),
	})
	svc := ecs.New(sess)

	dp := Deployment{svc, rtm, msg, service, tag, map[string]bool{}}
	dp.postMessage("Deployment started…")

	// Find the service name, an exact match is not possible
	serviceName, err := dp.getServiceName()

	// Create new task definition
	newTask, err := dp.getTaskDefinition()
	if err != nil {
		return err
	}

	newTaskDef, err := svc.RegisterTaskDefinition(&newTask)
	if err != nil {
		return err
	}

	descServiceInput := &ecs.DescribeServicesInput{
		Cluster: aws.String(Config.Cluster),
		Services: []*string{
			aws.String(serviceName),
		},
	}

	dp.checkMessages(descServiceInput, false)

	// Update the service with the new Task Definition
	taskDefName := fmt.Sprintf("%s:%d", *newTaskDef.TaskDefinition.Family, *newTaskDef.TaskDefinition.Revision)
	if _, err = svc.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        aws.String(Config.Cluster),
		DesiredCount:   aws.Int64(1),
		Service:        aws.String(serviceName),
		TaskDefinition: aws.String(taskDefName),
	}); err != nil {
		return err
	}

	waitChan := make(chan error)
	timerChan := time.NewTicker(2 * time.Second)
	startTime := time.Now()

	go func(waitChan chan error) {
		before := time.Now()
		waitChan <- svc.WaitUntilServicesStable(descServiceInput)
		dp.postMessage(":timer_clock: %s", time.Since(before))
	}(waitChan)

	dp.checkMessages(descServiceInput, true)
	exitLoop := false
	for !exitLoop {
		select {
		// Wait for the new version of the service to be entirely rolled out
		case err = <-waitChan:
			if err != nil {
				return err
			}
			exitLoop = true

		case <-timerChan.C:
			// If we are still waiting, output the log messages and make sure we aren't over the time limit.
			if time.Since(startTime) > TimeLimit {
				dp.postMessage(":red_circle: Deployment timeout of %s exceeded", TimeLimit)
				//exitLoop = true
			}
			dp.checkMessages(descServiceInput, true)
		}
	}

	dp.postMessage(":white_check_mark:")
	return nil
}

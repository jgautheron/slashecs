package deployer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	. "github.com/jgautheron/slashecs/config"
	"github.com/nlopes/slack"
)

type Deployment struct {
	svc          *ecs.ECS
	rtm          *slack.RTM
	msg          *slack.MessageEvent
	service, tag string
	seenMessages map[string]bool
}

func (d Deployment) getTaskDefinition() (ecs.RegisterTaskDefinitionInput, error) {
	// Retrieve the current task definition
	task, err := d.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(d.service),
	})
	if err != nil {
		return ecs.RegisterTaskDefinitionInput{}, err
	}

	container := task.TaskDefinition.ContainerDefinitions[0]
	newContainer := container.SetImage(d.getUpdatedImage(*container.Image))
	return ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{newContainer},
		Family:               task.TaskDefinition.Family,
		NetworkMode:          task.TaskDefinition.NetworkMode,
		PlacementConstraints: task.TaskDefinition.PlacementConstraints,
		Volumes:              task.TaskDefinition.Volumes,
	}, nil
}

func (d Deployment) getUpdatedImage(image string) string {
	sp := strings.Split(image, ":")
	return strings.Join([]string{sp[0], d.tag}, ":")
}

func (d Deployment) getServiceName() (string, error) {
	resp, err := d.svc.ListServices(&ecs.ListServicesInput{
		Cluster:    aws.String(Config.Cluster),
		MaxResults: aws.Int64(100),
	})
	if err != nil {
		return "", err
	}

	for _, arn := range resp.ServiceArns {
		if strings.Contains(*arn, d.service) {
			return strings.Split(*arn, "/")[1], nil
		}
	}

	return "", errors.New("Service not found")
}

func (d Deployment) checkMessages(descServiceInput *ecs.DescribeServicesInput, writeToSTDOUT bool) {
	serviceDef, err := d.svc.DescribeServices(descServiceInput)
	if err != nil {
		logger.Error(err)
		return
	}

	for _, service := range serviceDef.Services {
		for _, event := range service.Events {
			if !d.seenMessages[*event.Id] {
				if writeToSTDOUT {
					d.postMessage(*event.Message)
				}
				d.seenMessages[*event.Id] = true
			}
		}

	}
}

func (d Deployment) postMessage(args ...interface{}) {
	message := args[0].(string)
	if len(args) > 1 {
		message = fmt.Sprintf(args[0].(string), args[1:]...)
	}
	d.rtm.SendMessage(d.rtm.NewOutgoingMessage(
		fmt.Sprintf("*[%s:%s]* %s", d.service, d.tag, message),
		d.msg.Channel,
	))
}

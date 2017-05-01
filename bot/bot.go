package bot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	. "github.com/jgautheron/slashecs/config"
	"github.com/nlopes/slack"
)

func New() {
	token := Config.SlackToken
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			// fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)

				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					respond(rtm, ev, prefix)
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				//Take no action
			}
		}
	}
}

func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string) {
	// var response string
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	r := regexp.MustCompile(`^deploy ([\w\d\-\_]+) ([\w\d\-\_]+)$`)

	// @deploybot deploy networking-client 235

	matches := r.FindStringSubmatch(text)
	if len(matches) > 0 {
		fmt.Println("deploy")
		deploy(matches[1], matches[2])
	}

	// if acceptedGreetings[text] {
	// 	response = "What's up buddy!?!?!"
	// 	rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	// } else if acceptedHowAreYou[text] {
	// 	response = "Good. How are you?"
	// 	rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	// }
}

func deploy(service, tag string) {
	sess := session.New(&aws.Config{
		Region: aws.String(Config.AwsRegion),
		Credentials: credentials.NewStaticCredentials(
			Config.AwsAccessKeyID,
			Config.AwsSecretAccessKey,
			"",
		),
	})
	svc := ecs.New(sess)

	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(service),
	}
	resp, err := svc.DescribeTaskDefinition(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)

	// newTask := getTaskDefinition(resp, tag)
	// _, err = svc.RegisterTaskDefinition(&newTask)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	panic(err)
	// }

}

func getTaskDefinition(task *ecs.DescribeTaskDefinitionOutput, tag string) ecs.RegisterTaskDefinitionInput {
	container := task.TaskDefinition.ContainerDefinitions[0]
	newContainer := container.SetImage(getUpdatedImage(*container.Image, tag))
	return ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{newContainer},
		Family:               task.TaskDefinition.Family,
		NetworkMode:          task.TaskDefinition.NetworkMode,
		PlacementConstraints: task.TaskDefinition.PlacementConstraints,
		Volumes:              task.TaskDefinition.Volumes,
	}
}

func getUpdatedImage(image, tag string) string {
	sp := strings.Split(":", image)
	return strings.Join([]string{sp[0], tag}, ":")
}

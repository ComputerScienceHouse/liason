package main

import (
	"fmt"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
)

var LiasonChannel = os.Getenv("LIASON_CHANNEL_ID")

// [eboard] client
var EboardRelations = make(map[string]string)

// [client] eboard
var ClientRelations = make(map[string]string)

// [handle] id
var UserGroups = make(map[string]string)

// just handles
var UserGroupHandles = make([]string, 0)

const ThisUserID = "U08PJ68RKLN"

func main() {
	if LiasonChannel == "" {
		fmt.Fprintf(os.Stderr, "LIASON_CHANNEL_ID must be set.\n")
		os.Exit(1)
	}

	//get and verify app token (TBH this might be removable)
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		fmt.Fprintf(os.Stderr, "SLACK_APP_TOKEN must be set.\n")
		os.Exit(1)
	}
	if !strings.HasPrefix(appToken, "xapp-") {
		fmt.Fprintf(os.Stderr, "SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	//get and verify bot token
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		fmt.Fprintf(os.Stderr, "SLACK_BOT_TOKEN must be set.\n")
		os.Exit(1)
	}
	if !strings.HasPrefix(botToken, "xoxb-") {
		fmt.Fprintf(os.Stderr, "SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	//load API accordingly
	api := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	client := socketmode.New(api) //,
	//	socketmode.OptionDebug(true),
	//	socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)))
	socketHandler := socketmode.NewSocketmodeHandler(client)

	//handle Slack normal endpoints
	socketHandler.Handle(socketmode.EventTypeConnecting, func(event *socketmode.Event, client *socketmode.Client) {
		fmt.Println("Connecting to Slack")
	})
	socketHandler.Handle(socketmode.EventTypeConnectionError, func(event *socketmode.Event, client *socketmode.Client) {
		fmt.Println("Connection Failed. Trying Again?")
	})
	socketHandler.Handle(socketmode.EventTypeConnected, func(event *socketmode.Event, client *socketmode.Client) {
		fmt.Println("Connected to Slack via Socket Mode.")
	})
	socketHandler.Handle(socketmode.EventTypeHello, func(event *socketmode.Event, client *socketmode.Client) {
		fmt.Println("Hello message received.")
	})

	LoadRelations()

	groups, err := client.GetUserGroups(slack.GetUserGroupsOptionIncludeUsers(false), slack.GetUserGroupsOptionIncludeDisabled(false))
	if err != nil {
		log.Panicln(err)
		return
	}
	for _, group := range groups {
		UserGroups[group.Handle] = group.ID
	}
	UserGroupHandles = slices.Collect(maps.Keys(UserGroups))

	socketHandler.Handle(socketmode.EventTypeEventsAPI, handleEvents)

	socketHandler.HandleSlashCommand("/start", startCommand)

	client.SetUserPresence("auto")
	socketHandler.RunEventLoop()

}

func handleEvents(evt *socketmode.Event, client *socketmode.Client) {
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}
	client.Ack(*evt.Request)

	if eventsAPIEvent.Type != slackevents.CallbackEvent {
		return
	}
	if eventsAPIEvent.InnerEvent.Type != "message" {
		return
	}
	event := eventsAPIEvent.InnerEvent.Data.(*slackevents.MessageEvent)
	if event.User == ThisUserID || event.User == "" {
		return
	}
	switch event.ChannelType {
	case "group":
		handleEboard(event, client)
	case "im":
		handleDM(event, client)
	default:
		return
	}
}

func handleEboard(event *slackevents.MessageEvent, client *socketmode.Client) {
	if event.Channel != LiasonChannel {
		fmt.Println("Messaged from a different channel?")
		client.LeaveConversation(event.Channel)
		return
	}
	if event.ThreadTimeStamp == "" {
		return
	}
	eboardTS := EboardRelations[event.ThreadTimeStamp]
	if eboardTS == "" {
		return
	}
	sendData := strings.Split(eboardTS, ":")
	client.AddReaction("circle-game", slack.ItemRef{Channel: event.Channel, Timestamp: event.TimeStamp})
	client.SendMessage(sendData[0], slack.MsgOptionUsername("EBoard"), slack.MsgOptionTS(sendData[1]), slack.MsgOptionText(event.Text, false))
}

func handleDM(event *slackevents.MessageEvent, client *socketmode.Client) {
	if event.ThreadTimeStamp == "" {
		client.SendMessage(event.Channel, slack.MsgOptionText(
			"Want to start a new conversation? /start _group/subgroups_... _Subject of the conversation_\n"+
				"Groups/subgroups could be eboard, socials, opchom, etc. Just make sure it's a real user group in Slack", false))
		return
	}
	sendTS := ClientRelations[event.Channel+":"+event.ThreadTimeStamp]
	if sendTS == "" {
		return
	}
	client.AddReaction("circle-game", slack.ItemRef{Channel: event.Channel, Timestamp: event.TimeStamp})
	client.SendMessage(LiasonChannel, slack.MsgOptionUsername("Concerned Member"), slack.MsgOptionTS(sendTS), slack.MsgOptionText(event.Text, false))
	//TODO: maybe hash threadTS for privacy?
}

func startCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return
	}
	client.Ack(*evt.Request)

	if cmd.ChannelName != "directmessage" {
		return
	}
	split := strings.Split(cmd.Text, " ")
	index := 0
	mentions := ""
	for i, str := range split {
		if !slices.Contains(UserGroupHandles, str) {
			index = i
			break
		}
		mentions += "<!subteam^" + UserGroups[str] + "> "
	}

	subject := strings.Join(split[index:], " ")

	_, dmTS, _, _ := client.SendMessage(cmd.ChannelID, slack.MsgOptionText("Conversation :thread: here <@"+cmd.UserID+"> about "+subject, false))
	_, ebTS, _, _ := client.SendMessage(LiasonChannel, slack.MsgOptionText("New :thread: for "+mentions+": "+subject, false))
	clientStr := cmd.ChannelID + ":" + dmTS
	EboardRelations[ebTS] = clientStr
	ClientRelations[clientStr] = ebTS
	MakeRelation(ebTS, cmd.ChannelID, dmTS)
}

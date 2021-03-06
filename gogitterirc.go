package main

import (
	"./go-gitter"
	"fmt"

	"github.com/jinzhu/configor"
	"github.com/thoj/go-ircevent"
)

type Config struct {
	IRC struct {
		Server  string `default:"irc.freenode.net:6667"`
		Nick    string `required:"true"`
		Channel string `required:"true"`
	}
	Gitter struct {
		Apikey string `required:"true"`
		Room   string `required:"true"`
	}
}

func main() {
	fmt.Println("Gitter/IRC Sync Bot, written in Go by mrexodia")
	var conf Config
	if err := configor.Load(&conf, "config.json"); err != nil {
		fmt.Printf("Error loading config: %v...\n", err)
		return
	}

	api := gitter.New(conf.Gitter.Apikey)
	api.SetDebug(true, nil)
	user, err := api.GetUser()
	if err != nil {
		fmt.Printf("GetUser error: %v\n", err)
		return
	}
	room, err := api.JoinRoom(conf.Gitter.Room)
	if err != nil {
		fmt.Printf("JoinRoom error: %v\n", err)
		return
	}
	fmt.Printf("Joined room with ID: %v\n", room.ID)

	ircCon := irc.IRC(conf.IRC.Nick, conf.IRC.Nick)
	if err := ircCon.Connect(conf.IRC.Server); err != nil {
		fmt.Printf("Failed to connect to %v: %v...\n", conf.IRC.Server, err)
		return
	}
	ircCon.AddCallback("001", func(e *irc.Event) {
		ircCon.Join(conf.IRC.Channel)
	})
	ircCon.AddCallback("JOIN", func(e *irc.Event) {
		ircCon.Privmsg(conf.IRC.Channel, "Hello, I'll be syncronizing between IRC and Gitter today!")
	})
	ircCon.AddCallback("PRIVMSG", func(e *irc.Event) {
		gitterMsg := fmt.Sprintf("<%v> %v", e.Nick, e.Message())
		fmt.Printf("[IRC] %v\n", gitterMsg)
		api.SendMessage(room.ID, gitterMsg)
	})
	go ircCon.Loop()

	stream := api.Stream(room.ID)
	go api.Listen(stream)

	for {
		event := <-stream.Event
		switch ev := event.Data.(type) {
		case *gitter.MessageReceived:
			if ev.Message.From.Username != user.Username {
				ircMsg := fmt.Sprintf("<%v> %v", ev.Message.From.Username, ev.Message.Text)
				fmt.Printf("[Gitter] %v\n", ircMsg)
				ircCon.Privmsg(conf.IRC.Channel, ircMsg)
			}
		case *gitter.GitterConnectionClosed:
			fmt.Printf("[Gitter] Connection closed...\n")
		}
	}
}

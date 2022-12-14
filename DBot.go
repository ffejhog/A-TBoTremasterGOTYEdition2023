package main

import "C"
import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
)

type Config struct {
	DiscordSessionToken string `env:"DISCORD_SESSION_TOKEN" flag:"sessionToken" flagDesc:"The Session token for the Discord Bot"`
	DatabaseDir         string `env:"DATABASE_DIRECTORY" flag:"dbDir" flagDesc:"The directory to store the database in"`
	DefaultMessage      string `env:"DEFAULT_MESSAGE" flag:"defaultMessage" flagDesc:"The default response"`
}

type DBot struct {
	Discord          *discordgo.Session
	Config           Config
	DB               *clover.DB
	MarkovCollection string
	DefaultMessage   string
}

func (b *DBot) Connect() error {
	var err error
	if b.Config.DatabaseDir == "" {
		b.Config.DatabaseDir = "DBotDatabase"
	}
	b.DB, _ = clover.Open(b.Config.DatabaseDir)
	collectionExists, _ := b.DB.HasCollection(b.MarkovCollection)
	if !collectionExists {
		b.DB.CreateCollection(b.MarkovCollection)
	}

	b.Discord, err = discordgo.New("Bot " + b.Config.DiscordSessionToken)
	if err != nil {
		fmt.Println("error opening connection,", err)
	}
	// Handlers
	b.Discord.AddHandler(b.TrainMarkov)
	b.Discord.AddHandler(b.RespondMarkov)

	b.Discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = b.Discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
	}
	return err
}

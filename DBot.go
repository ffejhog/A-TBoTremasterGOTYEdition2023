package main

import "C"
import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
	"log"
)

type Config struct {
	DiscordSessionToken string `env:"DISCORD_SESSION_TOKEN" flag:"sessionToken" flagDesc:"The Session token for the Discord Bot"`
	DatabaseDir         string `env:"DATABASE_DIRECTORY" flag:"dbDir" flagDesc:"The directory to store the database in"`
	RenderAPIKey        string `env:"COMPUTER_RENDER_API_TOKEN" flag:"computerrenderapitoken" flagDesc:"The Computer Render api token for image generation"`
}

type DBot struct {
	Discord          *discordgo.Session
	Config           Config
	DB               *clover.DB
	MarkovCollection string
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

	dbDumpCommand := discordgo.ApplicationCommand{
		Name:        "dbdump",
		Description: "Completely Dumps the Markov Chain Database",
	}

	imageGenCommand := discordgo.ApplicationCommand{
		Name:        "give",
		Description: "Generate an AI image",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "Prompt",
				Required:    true,
			}},
	}

	_, err = b.Discord.ApplicationCommandCreate(b.Discord.State.User.ID, "", &dbDumpCommand)
	if err != nil {
		log.Panicf("Cannot create command: %v", err)
	}

	_, err = b.Discord.ApplicationCommandCreate(b.Discord.State.User.ID, "", &imageGenCommand)
	if err != nil {
		log.Panicf("Cannot create command: %v", err)
	}

	b.Discord.AddHandler(b.DumpDatabase)
	b.Discord.AddHandler(b.GenerateImage)

	return err
}

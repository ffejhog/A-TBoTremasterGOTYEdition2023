package models

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
)

type Config struct {
	DiscordSessionToken string `env:"DISCORD_SESSION_TOKEN" flag:"sessionToken" flagDesc:"The Session token for the Discord Bot"`
	DatabaseDir         string `env:"DATABASE_DIRECTORY" flag:"dbDir" flagDesc:"The directory to store the database in"`
}

type DBot struct {
	Discord *discordgo.Session

	Config Config
	DB     *clover.DB
}

func (b *DBot) Connect() error {
	// Handlers
	b.Discord.AddHandler(TrainMarkov)

	b.Discord.Identify.Intents = discordgo.IntentsGuildMessages

	err := b.Discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
	}
}

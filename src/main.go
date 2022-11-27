package main

import (
	"DongBotRemastered/src/handlers"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ian-kent/gofigure"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	DiscordSessionToken string `env:"DISCORD_SESSION_TOKEN" flag:"sessionToken" flagDesc:"The Session token for the Discord Bot"`
	DatabaseDir         string `env:"DATABASE_DIRECTORY" flag:"dbDir" flagDesc:"The directory to store the database in"`
}

func main() {
	var cfg Config
	err := gofigure.Gofigure(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	discord, err := discordgo.New("Bot " + cfg.DiscordSessionToken)

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Handlers
	discord.AddHandler(handlers.TrainMarkov)

	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	discord.Close()
}

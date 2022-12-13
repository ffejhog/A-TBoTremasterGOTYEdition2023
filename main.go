package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ffejhog/A-TBoTremasterGOTYEdition2023/src/models"
	"github.com/ian-kent/gofigure"
	"github.com/ostafen/clover"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var cfg Config
	err := gofigure.Gofigure(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	db, _ := clover.Open(cfg.DatabaseDir)
	collectionExists, _ := db.HasCollection("markovStorage")
	if !collectionExists {
		db.CreateCollection("markovStorage")
	}
	defer db.Close()

	discord, err := discordgo.New("Bot " + cfg.DiscordSessionToken)

	dBot := models.DBot{
		DB:     db,
		Config: cfg,
	}

	dBot.Connect()

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dBot.Discord.Close()
}

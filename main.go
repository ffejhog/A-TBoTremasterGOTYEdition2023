package main

import (
	"fmt"
	"github.com/ian-kent/gofigure"
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

	dBot := DBot{
		Config:           cfg,
		MarkovCollection: "markovStorage",
	}

	err = dBot.Connect()
	if err != nil {
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	dBot.DB.Close()
	dBot.Discord.Close()
}

package main

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
	"math/rand"
	"os"
	"strings"
	"time"
)

type markovTrainingRecord struct {
	StartWord  string `clover:"startWord"`
	FollowWord string `clover:"followWord"`
	CanBeFirst bool   `clover:"canBeFirst"`
	CanBeLast  bool   `clover:"canBeLast"`
}

func (b *DBot) TrainMarkov(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if containsUser(m.Content, "<@"+s.State.User.ID+">") {
		return
	}

	var documents []*clover.Document
	words := strings.Fields(m.Content)

	if len(words) < 4 {
		return
	}

	for i := 0; i < len(words); i++ {
		var sentenceStart bool
		var sentenceEnd bool
		if i == 0 {
			sentenceStart = true
		} else {
			sentenceStart = false
		}
		if strings.HasSuffix(words[i], ".") {
			sentenceEnd = true
		} else {
			sentenceEnd = false
		}
		var trainElement markovTrainingRecord
		if i == len(words)-1 {
			trainElement = markovTrainingRecord{
				StartWord:  words[i],
				FollowWord: "",
				CanBeFirst: sentenceStart,
				CanBeLast:  true,
			}
		} else {
			trainElement = markovTrainingRecord{
				StartWord:  words[i],
				FollowWord: words[i+1],
				CanBeFirst: sentenceStart,
				CanBeLast:  sentenceEnd,
			}
		}

		fmt.Println(trainElement)
		documents = append(documents, clover.NewDocumentOf(trainElement))
	}

	b.DB.Insert(b.MarkovCollection, documents...)
}

func (b *DBot) RespondMarkov(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if !containsUser(m.Content, "<@"+s.State.User.ID+">") {
		return
	}
	fmt.Println("Responding...")
	var generatedResponse bytes.Buffer
	rand.Seed(time.Now().Unix())
	searchedRecords, _ := b.DB.Query(b.MarkovCollection).Where(clover.Field("canBeFirst").IsTrue()).FindAll()

	currentRecord := searchedRecords[rand.Intn(len(searchedRecords))]
	chainedWord := currentRecord.Get("followWord").(string)
	finalWord := false

	generatedResponse.WriteString(currentRecord.Get("startWord").(string))
	generatedResponse.WriteString(" ")
	for !finalWord {
		searchedRecords, _ := b.DB.Query(b.MarkovCollection).Where(clover.Field("startWord").Eq(chainedWord)).FindAll()
		if len(searchedRecords) > 0 {
			currentRecord = searchedRecords[rand.Intn(len(searchedRecords))]
			finalWord = currentRecord.Get("canBeLast").(bool)
			generatedResponse.WriteString(currentRecord.Get("startWord").(string))
			generatedResponse.WriteString(" ")
			chainedWord = currentRecord.Get("followWord").(string)
		} else {
			finalWord = true
		}
	}
	s.ChannelMessageSend(m.ChannelID, generatedResponse.String())
}

func (b *DBot) DumpDatabase(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "dbdump" {
		return
	}
	b.DB.ExportCollection(b.MarkovCollection, "export.json")

	file, _ := os.Open("export.json")

	discordAttachment := discordgo.File{
		Name:        "DBDump.json",
		ContentType: "application/json",
		Reader:      file,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Here is the Database dump you requested",
			Files:   []*discordgo.File{&discordAttachment},
		},
	})
}

func containsUser(s string, user string) bool {
	return strings.Contains(s, user)
}

package main

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
	"math/rand"
	"net/http"
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

func (b *DBot) GenerateImage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "give" {
		return
	}

	errMsg := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Generating...",
		},
	})

	fmt.Println(errMsg)

	prompt := strings.ReplaceAll(i.ApplicationCommandData().Options[0].Value.(string), " ", "-")

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://api.computerender.com/generate/"+prompt, nil)
	// ...
	req.Header.Add("Authorization", "X-API-Key "+b.Config.RenderAPIKey)
	resp, err := client.Do(req)

	if err != nil || resp.StatusCode >= 300 {
		failureString := "Failed to generate image :("
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &failureString,
		})
		return
	}

	discordAttachment := discordgo.File{
		Name:        prompt + ".jpeg",
		ContentType: "image/jpeg",
		Reader:      resp.Body,
	}

	responseString := "Here is your " + i.ApplicationCommandData().Options[0].Value.(string)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &responseString,
		Files:   []*discordgo.File{&discordAttachment},
	})
	fmt.Println(err)
	defer resp.Body.Close()
}

func containsUser(s string, user string) bool {
	return strings.Contains(s, user)
}

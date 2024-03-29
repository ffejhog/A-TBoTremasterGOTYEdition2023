package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/donovanhide/eventsource"
	"github.com/ostafen/clover"
	"github.com/sashabaranov/go-openai"
	"math/rand"
	"net/http"
	"net/url"
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

func (b *DBot) RespondGPT(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if !containsUser(m.Content, "<@"+s.State.User.ID+">") {
		return
	}
	fmt.Println("Responding...")

	lastMessages, _ := s.ChannelMessages(m.ChannelID, 20, "", "", "")

	var gptMessages []openai.ChatCompletionMessage
	gptMessages = append(gptMessages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleSystem,
		Content: "You are a chatbot named " + b.Config.Name + ". Your responses are typically witty or humorous in some way." +
			"You are happy to respond to inquiries, and will fall back to humor whenever you are unsure what to respond." +
			"You are allowed to be mean to people within the channel, as we all consent to any targeted humor." +
			"Humor and wittiness is always your priority over kindness." +
			"The channel you are in typically is gaming related, but not always. The last few messages sent will be sent in the assistant role for context.",
	})

	for _, element := range lastMessages {
		gptMessages = append(gptMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: element.Content,
		})
	}

	gptMessages = append(gptMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: m.Content,
	})

	resp, err := b.GPT.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Davinci,
			Messages: gptMessages,
		},
	)

	fmt.Println("Logging Response:")
	fmt.Println(resp)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	s.ChannelMessageSend(m.ChannelID, resp.Choices[0].Message.Content)
}

func (b *DBot) RespondLlama(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !containsUser(m.Content, "<@"+s.State.User.ID+">") {
		return
	}
	generatedPrompt := strings.ReplaceAll(m.Content, "<@"+s.State.User.ID+">", "")
	genUrl := b.Config.SergeApiEndpoint + b.Config.SergeChatId + "/question?prompt=" + url.QueryEscape(generatedPrompt)
	fmt.Println(genUrl)
	stream, err := eventsource.Subscribe(genUrl, "")
	if err != nil {
		panic(err)
	}
	builder := strings.Builder{}
builderLoop:
	for {
		select {
		case ev := <-stream.Events:
			fmt.Println(ev.Id(), ev.Event(), ev.Data())
			if ev.Event() == "message" {
				builder.WriteString(ev.Data())
			}
		case <-time.After(16 * time.Second):
			break builderLoop
		}
	}
	fmt.Println("Message complete")
	if builder.Len() == 0 {
		s.ChannelMessageSend(m.ChannelID, "Failed to generate response. Check chat instance id.")
	} else {
		s.ChannelMessageSend(m.ChannelID, builder.String())
	}
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

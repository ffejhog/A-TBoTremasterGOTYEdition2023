package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ostafen/clover"
	"strings"
)

type markovTrainingRecord struct {
	StartWord  string
	FollowWord string
	CanBeFirst bool
}

func (b *DBot) TrainMarkov(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	var documents []*clover.Document
	words := strings.Fields(m.Content)

	for i := 0; i < len(words)-1; i++ {
		var sentenceStart bool
		if i == 0 {
			sentenceStart = true
		} else {
			sentenceStart = false
		}

		trainElement := markovTrainingRecord{
			StartWord:  words[i],
			FollowWord: words[i+1],
			CanBeFirst: sentenceStart,
		}
		documents = append(documents, clover.NewDocumentOf(trainElement))
	}

	b.DB.Insert(b.MarkovCollection, documents...)
}

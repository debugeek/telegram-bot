package tgbot

import (
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Session struct {
	ID      int64
	User    *User
	command string
	client  *Client
}

func (s *Session) SendText(text string) {
	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text: text,
	}
	s.sendMessage(message)
}

func (s *Session) SendTextUsingParseMode(text string, parseMode string) {
	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text:      text,
		ParseMode: parseMode,
	}
	s.sendMessage(message)
}

func (s *Session) SendImage(file *os.File, name string) error {
	message := tgbotapi.NewPhotoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.sendMessage(message)
}

func (s *Session) SendVideo(file *os.File, name string) error {
	message := tgbotapi.NewVideoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.sendMessage(message)
}

func (s *Session) SendAudio(file *os.File, name string) error {
	message := tgbotapi.NewAudioUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.sendMessage(message)
}

func (s *Session) SendFile(file *os.File, name string) error {
	message := tgbotapi.NewDocumentUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.sendMessage(message)
}

func (s *Session) SendFormattedText(text string, promptKey string) {
	if text == "" {
		return
	}

	if promptText := s.client.CCMS.Texts.Prompts[promptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	s.SendText(text)
}

func (s *Session) SendFormattedTextUsingParseMode(text string, promptKey string, parseMode string) {
	if text == "" {
		return
	}

	if promptText := s.client.CCMS.Texts.Prompts[promptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	s.SendTextUsingParseMode(text, parseMode)
}

func (s *Session) sendMessage(message tgbotapi.Chattable) error {
	_, err := s.client.BotAPI.Send(message)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session) processError(err error) {
	switch err.Error() {
	case errChatNotFound, errNotMember:
		s.User.Blocked = true
		s.client.Firebase.updateUser(s.User)
	}
}

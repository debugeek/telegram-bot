package tgbot

import (
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Session[USERDATA any] struct {
	ID      int64
	User    *User[USERDATA]
	command string
	client  *Client[USERDATA]
}

func (s *Session[USERDATA]) SendText(text string) error {
	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text: text,
	}
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendTextUsingParseMode(text string, parseMode string) error {
	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text:      text,
		ParseMode: parseMode,
	}
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) ReplyText(text string, replyToMessageID int) error {
	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: replyToMessageID,
		},
		Text: text,
	}
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendImage(file *os.File, name string) error {
	message := tgbotapi.NewPhotoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendVideo(file *os.File, name string) error {
	message := tgbotapi.NewVideoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendAudio(file *os.File, name string) error {
	message := tgbotapi.NewAudioUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendFile(file *os.File, name string) error {
	message := tgbotapi.NewDocumentUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[USERDATA]) SendFormattedText(text string, promptKey string) error {
	if text == "" {
		return nil
	}

	if promptText := s.client.CCMS.Texts.Prompts[promptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	return s.SendText(text)
}

func (s *Session[USERDATA]) SendFormattedTextUsingParseMode(text string, promptKey string, parseMode string) error {
	if text == "" {
		return nil
	}

	if promptText := s.client.CCMS.Texts.Prompts[promptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	return s.SendTextUsingParseMode(text, parseMode)
}

func (s *Session[USERDATA]) SendMessage(message tgbotapi.Chattable) error {
	_, err := s.client.BotAPI.Send(message)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[USERDATA]) processError(err error) {
	switch err.Error() {
	case errChatNotFound, errNotMember:
		s.User.Blocked = true
		s.client.Firebase.UpdateUser(s.User)
	}
}

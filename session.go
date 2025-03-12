package tgbot

import (
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type MessageConfig struct {
	ReplyToMessageID int
	ParseMode        string
	PromptKey        string
}

type Session[USERDATA any] struct {
	ID             int64
	User           *User[USERDATA]
	CommandSession *CommandSession
	client         *Client[USERDATA]
}

func newSession[USERDATA any](user *User[USERDATA], client *Client[USERDATA]) *Session[USERDATA] {
	return &Session[USERDATA]{
		ID:   user.ID,
		User: user,
		CommandSession: &CommandSession{
			Args: make(map[string]any),
		},
		client: client,
	}
}

func (s *Session[USERDATA]) SendText(text string) error {
	return s.SendTextWithConfig(text, MessageConfig{})
}

func (s *Session[USERDATA]) ReplyText(text string, replyToMessageID int) error {
	return s.SendTextWithConfig(text, MessageConfig{
		ReplyToMessageID: replyToMessageID,
	})
}

func (s *Session[USERDATA]) SendTextWithConfig(text string, config MessageConfig) error {
	if promptText := s.client.CCMS.Texts.Prompts[config.PromptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text:      text,
		ParseMode: config.ParseMode,
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

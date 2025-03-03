package tgbot

import (
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type ParseMode int

const (
	ParseModePlain ParseMode = iota
	ParseModeHTML
	ParseModeMarkdown
)

type MessageConfig struct {
	ReplyToMessageID int
	ParseMode        ParseMode
	PromptKey        string
}

type Session[BOTDATA, USERDATA any] struct {
	ID             int64
	User           *User[USERDATA]
	CommandSession *CommandSession
	client         *Client[BOTDATA, USERDATA]
}

func newSession[BOTDATA any, USERDATA any](user *User[USERDATA], client *Client[BOTDATA, USERDATA]) *Session[BOTDATA, USERDATA] {
	return &Session[BOTDATA, USERDATA]{
		ID:   user.ID,
		User: user,
		CommandSession: &CommandSession{
			Args: make(map[string]any),
		},
		client: client,
	}
}

func (s *Session[BOTDATA, USERDATA]) SendText(text string) error {
	return s.SendTextWithConfig(text, MessageConfig{})
}

func (s *Session[BOTDATA, USERDATA]) ReplyText(text string, replyToMessageID int) error {
	return s.SendTextWithConfig(text, MessageConfig{
		ReplyToMessageID: replyToMessageID,
	})
}

func (s *Session[BOTDATA, USERDATA]) SendTextWithConfig(text string, config MessageConfig) error {
	if promptText := s.client.Preference.Texts.Prompts[config.PromptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           s.ID,
			ReplyToMessageID: 0,
		},
		Text: text,
	}

	switch config.ParseMode {
	case ParseModeHTML:
		message.ParseMode = tgbotapi.ModeHTML
	case ParseModeMarkdown:
		message.ParseMode = tgbotapi.ModeMarkdown
	default:
		break
	}

	return s.SendMessage(message)
}

func (s *Session[BOTDATA, USERDATA]) SendImage(file *os.File, name string) error {
	message := tgbotapi.NewPhotoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[BOTDATA, USERDATA]) SendVideo(file *os.File, name string) error {
	message := tgbotapi.NewVideoUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[BOTDATA, USERDATA]) SendAudio(file *os.File, name string) error {
	message := tgbotapi.NewAudioUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[BOTDATA, USERDATA]) SendFile(file *os.File, name string) error {
	message := tgbotapi.NewDocumentUpload(s.User.ID, tgbotapi.FileReader{
		Name:   name,
		Reader: file,
		Size:   -1,
	})
	return s.SendMessage(message)
}

func (s *Session[BOTDATA, USERDATA]) SendMessage(message tgbotapi.Chattable) error {
	_, err := s.client.BotAPI.Send(message)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) processError(err error) {
	switch err.Error() {
	case errChatNotFound, errNotMember:
		s.User.Blocked = true
		s.client.Firebase.UpdateUser(s.User)
	}
}

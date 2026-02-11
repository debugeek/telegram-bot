package tgbot

import (
	"context"
	"errors"
	"os"
	"strings"
)

type MessageConfig struct {
	ReplyToMessageID int
	ParseMode        ParseMode
	PromptKey        string
	ReplyMarkup      ReplyMarkup
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

	opts := &SendMessageOpts{
		ReplyToMessageID: config.ReplyToMessageID,
		ParseMode:        config.ParseMode,
		ReplyMarkup:      config.ReplyMarkup,
	}
	err := s.client.bot.SendMessage(context.Background(), s.ID, text, opts)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) SendQuery(prompt string, options []string, handler func(*Session[BOTDATA, USERDATA], string)) error {
	markup := s.client.createPendingQuery(s.ID, options, handler)
	if markup == nil {
		return nil
	}
	return s.SendTextWithConfig(prompt, MessageConfig{
		ReplyMarkup: markup,
	})
}

func (s *Session[BOTDATA, USERDATA]) SendImage(file *os.File, name string) error {
	err := s.client.bot.SendPhoto(context.Background(), s.User.ID, file, name)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) SendVideo(file *os.File, name string, meta *VideoMeta) error {
	err := s.client.bot.SendVideo(context.Background(), s.User.ID, file, name, meta)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) SendAudio(file *os.File, name string) error {
	err := s.client.bot.SendAudio(context.Background(), s.User.ID, file, name)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) SendFile(file *os.File, name string) error {
	err := s.client.bot.SendDocument(context.Background(), s.User.ID, file, name)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) AnswerCallbackQuery(callbackQueryID string) error {
	err := s.client.bot.AnswerCallbackQuery(context.Background(), callbackQueryID)
	if err != nil {
		s.processError(err)
	}
	return err
}

func (s *Session[BOTDATA, USERDATA]) processError(err error) {
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrChatNotFound) {
		s.User.Blocked = true
		s.client.Firebase.UpdateUser(s.User)
	}
}

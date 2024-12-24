package tgbot

import (
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

func (s *Session) SendFormattedText(session *Session, text string, promptKey string) {
	if text == "" {
		return
	}

	if promptText := s.client.CCMS.Texts.Prompts[promptKey]; promptText != "" {
		text = strings.Join([]string{text, promptText}, "\n\n")
	}

	s.SendText(text)
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

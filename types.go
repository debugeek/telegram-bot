package tgbot

import (
	"context"
	"errors"
	"io"
	"strings"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrChatNotFound = errors.New("chat not found or bot is not a member")
)

type Update struct {
	Message       *Message
	CallbackQuery *CallbackQuery
}

type ParseMode int

const (
	ParseModePlain ParseMode = iota
	ParseModeHTML
	ParseModeMarkdown
)

type ReplyMarkup interface{}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

type SendMessageOpts struct {
	ReplyToMessageID int
	ParseMode        ParseMode
	ReplyMarkup      ReplyMarkup
}

type VideoMeta struct {
	Duration int
	Width    int
	Height   int
}

type BotAPI interface {
	SendMessage(ctx context.Context, chatID int64, text string, opts *SendMessageOpts) error
	SendPhoto(ctx context.Context, chatID int64, photo io.Reader, filename string) error
	SendVideo(ctx context.Context, chatID int64, video io.Reader, filename string, meta *VideoMeta) error
	SendAudio(ctx context.Context, chatID int64, audio io.Reader, filename string) error
	SendDocument(ctx context.Context, chatID int64, doc io.Reader, filename string) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string) error
}

type MessageSender struct {
	ID int64
}

type Chat struct {
	ID   int64
	Type string
}

func (c Chat) IsGroup() bool      { return c.Type == "group" }
func (c Chat) IsSuperGroup() bool { return c.Type == "supergroup" }

type Message struct {
	MessageID int
	Chat      Chat
	From      *MessageSender
	Text      string
}

type CallbackQuery struct {
	ID              string
	From            *MessageSender
	Message         *Message
	Data            string
	InlineMessageID string
}

func (m *Message) IsCommand() bool {
	return strings.HasPrefix(m.Text, "/")
}

func (m *Message) Command() string {
	if !m.IsCommand() {
		return ""
	}
	s := strings.TrimPrefix(m.Text, "/")
	if i := strings.IndexAny(s, " @"); i > 0 {
		s = s[:i]
	} else if i := strings.Index(s, " "); i > 0 {
		s = s[:i]
	}
	return s
}

func (m *Message) CommandArguments() string {
	if !m.IsCommand() {
		return ""
	}
	s := strings.TrimSpace(strings.TrimPrefix(m.Text, "/"))
	if i := strings.Index(s, " "); i >= 0 {
		return strings.TrimSpace(s[i+1:])
	}
	return ""
}

type User[USERDATA any] struct {
	ID       int64    `firestore:"id"`
	Blocked  bool     `firestore:"blocked"`
	UserData USERDATA `firestore:"userdata"`
}

type Preference[BOTDATA any] struct {
	Admins                      map[int64]string `json:"admins"`
	Texts                       Texts            `json:"texts"`
	BotData                     BOTDATA          `json:"botdata"`
	OnlyAdminsCanCommandInGroup bool             `json:"onlyAdminsCanCommandInGroup"`
}

type Texts struct {
	Prompts       map[string]string `json:"prompts"`
	Localizations map[string]string `json:"localizations"`
}

type CommandSession struct {
	Command string
	Stage   string
	Args    map[string]any
}

const (
	errChatNotFound = "Bad Request: chat not found"
	errNotMember    = "Forbidden: bot is not a member of the channel chat"
)

const (
	CmdStart     = "start"
	CmdBotReload = "botreload"
	CmdBotStat   = "botstat"
)

type CmdResult int

const (
	CmdResultProcessed CmdResult = iota
	CmdResultWaitingForInput
)

var GroupAnonymousBot = 1087968824

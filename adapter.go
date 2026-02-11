package tgbot

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type botImpl struct {
	b      *bot.Bot
	cancel context.CancelFunc
}

func newBotImpl(token string, onUpdate func(*Update)) (*botImpl, error) {
	b, err := bot.New(token, bot.WithDefaultHandler(func(ctx context.Context, _ *bot.Bot, raw *models.Update) {
		u := updateFromModels(raw)
		if u != nil {
			onUpdate(u)
		}
	}))
	if err != nil {
		return nil, err
	}
	return &botImpl{b: b}, nil
}

func updateFromModels(raw *models.Update) *Update {
	if raw == nil {
		return nil
	}

	if raw.CallbackQuery != nil {
		query := callbackQueryFromModels(raw.CallbackQuery)
		if query != nil {
			return &Update{CallbackQuery: query}
		}
	}

	var m *models.Message
	if raw.Message != nil {
		m = raw.Message
	} else if raw.ChannelPost != nil {
		m = raw.ChannelPost
	}
	if m == nil {
		return nil
	}
	return &Update{Message: messageFromModels(m)}
}

func messageFromModels(m *models.Message) *Message {
	if m == nil {
		return nil
	}
	msg := &Message{
		MessageID: m.ID,
		Chat:      Chat{ID: m.Chat.ID, Type: string(m.Chat.Type)},
		Text:      m.Text,
	}
	if m.From != nil {
		msg.From = &MessageSender{ID: m.From.ID}
	}
	return msg
}

func callbackQueryFromModels(q *models.CallbackQuery) *CallbackQuery {
	if q == nil {
		return nil
	}

	query := &CallbackQuery{
		ID:              q.ID,
		Data:            q.Data,
		InlineMessageID: q.InlineMessageID,
	}
	query.From = &MessageSender{ID: q.From.ID}

	if q.Message.Message != nil {
		query.Message = messageFromModels(q.Message.Message)
	} else if q.Message.InaccessibleMessage != nil {
		query.Message = &Message{
			MessageID: q.Message.InaccessibleMessage.MessageID,
			Chat: Chat{
				ID:   q.Message.InaccessibleMessage.Chat.ID,
				Type: string(q.Message.InaccessibleMessage.Chat.Type),
			},
		}
	}

	return query
}

func (bi *botImpl) Start(ctx context.Context) {
	bi.b.Start(ctx)
}

func (bi *botImpl) Stop() {
	if bi.cancel != nil {
		bi.cancel()
	}
}

func (bi *botImpl) setCancel(cancel context.CancelFunc) {
	bi.cancel = cancel
}

func convertParseMode(pm ParseMode) models.ParseMode {
	switch pm {
	case ParseModeHTML:
		return models.ParseModeHTML
	case ParseModeMarkdown:
		return models.ParseModeMarkdown
	default:
		return ""
	}
}

func convertReplyMarkup(markup ReplyMarkup) models.ReplyMarkup {
	switch m := markup.(type) {
	case nil:
		return nil
	case *InlineKeyboardMarkup:
		if m == nil {
			return nil
		}
		rows := make([][]models.InlineKeyboardButton, 0, len(m.InlineKeyboard))
		for _, row := range m.InlineKeyboard {
			btns := make([]models.InlineKeyboardButton, 0, len(row))
			for _, btn := range row {
				btns = append(btns, models.InlineKeyboardButton{
					Text:         btn.Text,
					URL:          btn.URL,
					CallbackData: btn.CallbackData,
				})
			}
			rows = append(rows, btns)
		}
		return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	case InlineKeyboardMarkup:
		return convertReplyMarkup(&m)
	default:
		return nil
	}
}

func mapSendError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, bot.ErrorForbidden) {
		return ErrForbidden
	}
	msg := err.Error()
	if strings.Contains(msg, errChatNotFound) || strings.Contains(msg, errNotMember) {
		return ErrChatNotFound
	}
	return err
}

func (bi *botImpl) SendMessage(ctx context.Context, chatID int64, text string, opts *SendMessageOpts) error {
	params := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}
	if opts != nil {
		params.ParseMode = convertParseMode(opts.ParseMode)
		if opts.ReplyToMessageID != 0 {
			params.ReplyParameters = &models.ReplyParameters{MessageID: opts.ReplyToMessageID}
		}
		params.ReplyMarkup = convertReplyMarkup(opts.ReplyMarkup)
	}
	_, err := bi.b.SendMessage(ctx, params)
	return mapSendError(err)
}

func (bi *botImpl) SendPhoto(ctx context.Context, chatID int64, photo io.Reader, filename string) error {
	_, err := bi.b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: chatID,
		Photo:  &models.InputFileUpload{Filename: filename, Data: photo},
	})
	return mapSendError(err)
}

func (bi *botImpl) SendVideo(ctx context.Context, chatID int64, video io.Reader, filename string, meta *VideoMeta) error {
	params := &bot.SendVideoParams{
		ChatID: chatID,
		Video:  &models.InputFileUpload{Filename: filename, Data: video},
	}
	if meta != nil {
		if meta.Duration > 0 {
			params.Duration = meta.Duration
		}
		if meta.Width > 0 {
			params.Width = meta.Width
		}
		if meta.Height > 0 {
			params.Height = meta.Height
		}
	}
	_, err := bi.b.SendVideo(ctx, params)
	return mapSendError(err)
}

func (bi *botImpl) SendAudio(ctx context.Context, chatID int64, audio io.Reader, filename string) error {
	_, err := bi.b.SendAudio(ctx, &bot.SendAudioParams{
		ChatID: chatID,
		Audio:  &models.InputFileUpload{Filename: filename, Data: audio},
	})
	return mapSendError(err)
}

func (bi *botImpl) SendDocument(ctx context.Context, chatID int64, doc io.Reader, filename string) error {
	_, err := bi.b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID:   chatID,
		Document: &models.InputFileUpload{Filename: filename, Data: doc},
	})
	return mapSendError(err)
}

func (bi *botImpl) AnswerCallbackQuery(ctx context.Context, callbackQueryID string) error {
	_, err := bi.b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: callbackQueryID})
	return mapSendError(err)
}

func chatMemberUserID(cm models.ChatMember) int64 {
	switch cm.Type {
	case models.ChatMemberTypeOwner:
		if cm.Owner != nil && cm.Owner.User != nil {
			return cm.Owner.User.ID
		}
	case models.ChatMemberTypeAdministrator:
		return cm.Administrator.User.ID
	case models.ChatMemberTypeMember:
		if cm.Member != nil && cm.Member.User != nil {
			return cm.Member.User.ID
		}
	case models.ChatMemberTypeRestricted:
		if cm.Restricted != nil && cm.Restricted.User != nil {
			return cm.Restricted.User.ID
		}
	case models.ChatMemberTypeLeft:
		if cm.Left != nil && cm.Left.User != nil {
			return cm.Left.User.ID
		}
	case models.ChatMemberTypeBanned:
		if cm.Banned != nil && cm.Banned.User != nil {
			return cm.Banned.User.ID
		}
	}
	return 0
}

func (bi *botImpl) GetChatAdministratorIDs(ctx context.Context, chatID int64) ([]int64, error) {
	admins, err := bi.b.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{ChatID: chatID})
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(admins))
	for _, a := range admins {
		if id := chatMemberUserID(a); id != 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

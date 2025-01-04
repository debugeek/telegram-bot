package tgbot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

type TgBot struct {
	Client *Client
}

type Config struct {
	TelegramBotToken    string
	FirebaseCredential  []byte
	FirebaseDatabaseURL string
}

func NewBot(config Config) *TgBot {
	client := newClient()
	client.initBotAPI(config.TelegramBotToken)
	client.initFirebase(config.FirebaseCredential, config.FirebaseDatabaseURL)
	return &TgBot{
		Client: client,
	}
}

func (tgbot *TgBot) RegisterMessageHandler(handler func(Session, tgbotapi.Message) bool) {
	tgbot.Client.registerMessageHandler(handler)
}

func (tgbot *TgBot) RegisterTextHandler(handler func(Session, string)) {
	tgbot.Client.registerTextHandler(handler)
}

func (tgbot *TgBot) RegisterReloadCommandHandler(handler func()) {
	tgbot.Client.registerReloadCommandHandler(handler)
}

func (tgbot *TgBot) RegisterCustomCommandHandler(cmd string, handler func(Session, string) bool) {
	tgbot.Client.registerCustomCommandHandler(cmd, handler)
}

func (tgbot *TgBot) Start() error {
	return tgbot.Client.start()
}

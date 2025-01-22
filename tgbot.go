package tgbot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

type TgBot[USERDATA any] struct {
	Client *Client[USERDATA]
}

type Config struct {
	TelegramBotToken    string
	FirebaseCredential  []byte
	FirebaseDatabaseURL string
}

func NewBot[USERDATA any](config Config) *TgBot[USERDATA] {
	client := newClient[USERDATA]()
	client.initBotAPI(config.TelegramBotToken)
	client.initFirebase(config.FirebaseCredential, config.FirebaseDatabaseURL)
	return &TgBot[USERDATA]{
		Client: client,
	}
}

func (tgbot *TgBot[USERDATA]) RegisterRawMessageHandler(handler func(Session[USERDATA], tgbotapi.Message) bool) {
	tgbot.Client.registerRawMessageHandler(handler)
}

func (tgbot *TgBot[USERDATA]) RegisterTextHandler(handler func(Session[USERDATA], string)) {
	tgbot.Client.registerTextHandler(handler)
}

func (tgbot *TgBot[USERDATA]) RegisterReloadCommandHandler(handler func()) {
	tgbot.Client.registerReloadCommandHandler(handler)
}

func (tgbot *TgBot[USERDATA]) RegisterCustomCommandHandler(cmd string, handler func(Session[USERDATA], string) bool) {
	tgbot.Client.registerCustomCommandHandler(cmd, handler)
}

func (tgbot *TgBot[USERDATA]) Start() error {
	return tgbot.Client.start()
}

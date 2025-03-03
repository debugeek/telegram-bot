package tgbot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

type TgBot[BOTDATA any, USERDATA any] struct {
	Client *Client[BOTDATA, USERDATA]
}

type Config struct {
	TelegramBotToken    string
	FirebaseCredential  []byte
	FirebaseDatabaseURL string
}

func NewBot[BOTDATA any, USERDATA any](config Config, delegate ClientDelegate[BOTDATA, USERDATA]) *TgBot[BOTDATA, USERDATA] {
	client := newClient(delegate)
	client.initBotAPI(config.TelegramBotToken)
	client.initFirebase(config.FirebaseCredential, config.FirebaseDatabaseURL)
	return &TgBot[BOTDATA, USERDATA]{
		Client: client,
	}
}

func (tgbot *TgBot[BOTDATA, USERDATA]) RegisterTextHandler(handler func(*Session[BOTDATA, USERDATA], string, *tgbotapi.Message)) {
	tgbot.Client.registerTextHandler(handler)
}

func (tgbot *TgBot[BOTDATA, USERDATA]) RegisterCommandHandler(cmd string, handler func(*Session[BOTDATA, USERDATA], string, *tgbotapi.Message) CmdResult) {
	tgbot.Client.registerCommandHandler(cmd, handler)
}

func (tgbot *TgBot[BOTDATA, USERDATA]) Start() error {
	return tgbot.Client.start()
}

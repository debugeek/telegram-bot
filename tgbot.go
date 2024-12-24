package tgbot

type TgBot struct {
	Client *Client
}

type TgBotConfig struct {
	BotToken            string
	FirebaseCredential  []byte
	FirebaseDatabaseURL string
}

func NewBot() *TgBot {
	return &TgBot{
		Client: &Client{
			Sessions:        make(map[int64]*Session),
			CommandHandlers: make(map[string]func(Session, string) bool),
		},
	}
}

func (tgbot *TgBot) RegisterTextHandler(handler func(Session, string) bool) {
	tgbot.Client.registerTextHandler(handler)
}

func (tgbot *TgBot) RegisterCommandHandler(cmd string, handler func(Session, string) bool) {
	tgbot.Client.registerCommandHandler(cmd, handler)
}

func (tgbot *TgBot) Start(config TgBotConfig) error {
	return tgbot.Client.start(config.BotToken, config.FirebaseCredential, config.FirebaseDatabaseURL)
}

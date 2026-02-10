package tgbot

type TgBot[BOTDATA any, USERDATA any] struct {
	Client *Client[BOTDATA, USERDATA]
}

type Config struct {
	TelegramBotToken    string
	FirebaseCredential  []byte
	FirebaseDatabaseURL string
}

func NewBot[BOTDATA any, USERDATA any](config Config, delegate ClientDelegate[BOTDATA, USERDATA]) (*TgBot[BOTDATA, USERDATA], error) {
	client, err := newClient(config, delegate)
	if err != nil {
		return nil, err
	}
	return &TgBot[BOTDATA, USERDATA]{
		Client: client,
	}, nil
}

func (tgbot *TgBot[BOTDATA, USERDATA]) RegisterTextHandler(handler func(*Session[BOTDATA, USERDATA], string, *Message)) {
	tgbot.Client.registerTextHandler(handler)
}

func (tgbot *TgBot[BOTDATA, USERDATA]) RegisterCommandHandler(cmd string, handler func(*Session[BOTDATA, USERDATA], string, *Message) CmdResult) {
	tgbot.Client.registerCommandHandler(cmd, handler)
}

func (tgbot *TgBot[BOTDATA, USERDATA]) Start() error {
	return tgbot.Client.start()
}

func (tgbot *TgBot[BOTDATA, USERDATA]) Stop() {
	tgbot.Client.stop()
}

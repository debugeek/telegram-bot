package tgbot

type User struct {
	ID      int64 `firestore:"id"`
	Blocked bool  `firestore:"blocked"`
}

type Texts struct {
	Prompts map[string]string `json:"prompts"`
}

type Config struct {
	BotToken string `json:"bot_token"`
}

type CCMS struct {
	Admins map[int64]string `json:"admins"`
	Config Config           `json:"config"`
	Texts  Texts            `json:"texts"`
}

const (
	errChatNotFound = "Bad Request: chat not found"
	errNotMember    = "Forbidden: bot is not a member of the channel chat"
)

const (
	CmdStart  = "start"
	CmdReload = "reload"
)
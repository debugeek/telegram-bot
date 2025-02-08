package tgbot

type User[USERDATA any] struct {
	ID       int64    `firestore:"id"`
	Blocked  bool     `firestore:"blocked"`
	UserData USERDATA `firestore:"userdata"`
}

type Texts struct {
	Prompts       map[string]string `json:"prompts"`
	Localizations map[string]string `json:"localizations"`
}

type CCMS struct {
	Admins map[int64]string `json:"admins"`
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

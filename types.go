package tgbot

type User[USERDATA any] struct {
	ID       int64    `firestore:"id"`
	Blocked  bool     `firestore:"blocked"`
	UserData USERDATA `firestore:"userdata"`
}

type Preference[BOTDATA any] struct {
	Admins  map[int64]string `json:"admins"`
	Texts   Texts            `json:"texts"`
	BotData BOTDATA          `json:"botdata"`
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

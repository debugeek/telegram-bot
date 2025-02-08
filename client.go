package tgbot

import (
	"context"
	"encoding/base64"
	"log"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"google.golang.org/api/option"
)

type ClientDelegate[USERDATA any] interface {
	DidLoadUser(*Session[USERDATA], *User[USERDATA])
}

type Handlers[USERDATA any] struct {
	RawMessageHandler     func(*Session[USERDATA], tgbotapi.Message) bool
	TextHandler           func(*Session[USERDATA], string)
	ReloadCommandHandler  func()
	CustomCommandHandlers map[string]func(*Session[USERDATA], string) bool
}

type Client[USERDATA any] struct {
	BotAPI   *tgbotapi.BotAPI
	Firebase Firebase[USERDATA]
	CCMS     CCMS
	Sessions map[int64]*Session[USERDATA]
	Handlers Handlers[USERDATA]
	mu       sync.RWMutex
	delegate ClientDelegate[USERDATA]
}

func newClient[USERDATA any](delegate ClientDelegate[USERDATA]) *Client[USERDATA] {
	return &Client[USERDATA]{
		Sessions: make(map[int64]*Session[USERDATA]),
		Handlers: Handlers[USERDATA]{
			CustomCommandHandlers: make(map[string]func(*Session[USERDATA], string) bool),
		},
		delegate: delegate,
	}
}

func (c *Client[USERDATA]) initFirebase(credential []byte, databaseURL string) error {
	context := context.Background()
	opt := option.WithCredentialsJSON(credential)
	conf := &firebase.Config{
		DatabaseURL: databaseURL,
	}
	app, err := firebase.NewApp(context, conf, opt)
	if err != nil {
		return err
	}

	firestore, err := app.Firestore(context)
	if err != nil {
		return err
	}

	database, err := app.Database(context)
	if err != nil {
		return err
	}

	firebase := Firebase[USERDATA]{
		Firestore: firestore,
		Database:  database,
		Context:   context,
	}
	c.Firebase = firebase

	return nil
}

func (c *Client[USERDATA]) initBotAPI(token string) error {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	botAPI.GetUpdates(u)

	c.BotAPI = botAPI

	return nil
}

func (c *Client[USERDATA]) start() error {
	users, err := c.Firebase.GetUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		session := &Session[USERDATA]{
			ID:     user.ID,
			User:   user,
			client: c,
		}
		c.insertSession(session)

		c.delegate.DidLoadUser(session, user)
	}

	c.reload()

	go c.runLoop()

	return nil
}

func (c *Client[USERDATA]) reload() error {
	var CCMS CCMS
	if err := c.Firebase.Database.NewRef("/ccms").Get(c.Firebase.Context, &CCMS); err != nil {
		return err
	}
	for key, val := range CCMS.Texts.Localizations {
		if bytes, err := base64.StdEncoding.DecodeString(val); err == nil {
			CCMS.Texts.Localizations[key] = string(bytes)
		}
	}
	for key, val := range CCMS.Texts.Prompts {
		if bytes, err := base64.StdEncoding.DecodeString(val); err == nil {
			CCMS.Texts.Prompts[key] = string(bytes)
		}
	}
	c.CCMS = CCMS

	if c.Handlers.ReloadCommandHandler != nil {
		c.Handlers.ReloadCommandHandler()
	}

	return nil
}

func (c *Client[USERDATA]) registerRawMessageHandler(handler func(*Session[USERDATA], tgbotapi.Message) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.RawMessageHandler = handler
}

func (c *Client[USERDATA]) registerTextHandler(handler func(*Session[USERDATA], string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.TextHandler = handler
}

func (c *Client[USERDATA]) registerReloadCommandHandler(handler func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.ReloadCommandHandler = handler
}

func (c *Client[USERDATA]) registerCustomCommandHandler(cmd string, handler func(*Session[USERDATA], string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.CustomCommandHandlers[cmd] = handler
}

func (c *Client[USERDATA]) runLoop() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10

	updates, err := c.BotAPI.GetUpdatesChan(u)
	if err != nil {
		log.Println(err)
		return
	}

	time.Sleep(time.Millisecond * 500)
	updates.Clear()

	for update := range updates {
		c.processUpdate(update)
	}
}

func (c *Client[USERDATA]) getSession(id int64) *Session[USERDATA] {
	return c.Sessions[id]
}

func (c *Client[USERDATA]) insertSession(session *Session[USERDATA]) {
	c.Sessions[session.ID] = session
}

func (c *Client[USERDATA]) processUpdate(update tgbotapi.Update) {
	var message *tgbotapi.Message
	if update.Message != nil {
		message = update.Message
	} else if update.ChannelPost != nil {
		message = update.ChannelPost
	}
	if message == nil {
		return
	}

	id := message.Chat.ID
	session := c.getSession(id)
	if session == nil {
		user := &User[USERDATA]{
			ID: id,
		}
		if err := c.Firebase.UpdateUser(user); err != nil {
			return
		}

		session = &Session[USERDATA]{
			ID:     id,
			User:   user,
			client: c,
		}
		c.insertSession(session)

		c.delegate.DidLoadUser(session, user)
	}

	c.processMessage(session, message)
}

func (c *Client[USERDATA]) processMessage(session *Session[USERDATA], message *tgbotapi.Message) {
	if session.User.Blocked {
		session.User.Blocked = false
		c.Firebase.UpdateUser(session.User)
	}

	if c.Handlers.RawMessageHandler != nil && c.Handlers.RawMessageHandler(session, *message) {
		return
	}

	if message.IsCommand() {
		switch message.Command() {
		case CmdStart:
			session.SendFormattedText("Greetings.", CmdStart)

		case CmdReload:
			if _, rv := c.CCMS.Admins[message.Chat.ID]; rv {
				c.reload()
				session.SendText("Done.")
			}

		default:
			c.processCommand(session, message.Command(), message.CommandArguments())
		}
	} else if session.command != "" {
		c.processCommand(session, session.command, message.Text)
	} else {
		c.processText(session, message.Text)
	}
}

func (c *Client[USERDATA]) processCommand(session *Session[USERDATA], command string, args string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler, exists := c.Handlers.CustomCommandHandlers[command]
	if !exists {
		return
	}

	if handler(session, args) {
		session.command = ""
	} else {
		session.command = command
	}
}

func (c *Client[USERDATA]) processText(session *Session[USERDATA], text string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler := c.Handlers.TextHandler
	if handler != nil {
		handler(session, text)
	}
}

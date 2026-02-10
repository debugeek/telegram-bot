package tgbot

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

type ClientDelegate[BOTDATA any, USERDATA any] interface {
	NewUserData() USERDATA
	DidLoadUser(*Session[BOTDATA, USERDATA], *User[USERDATA])
	DidLoadPreference()
}

type Handlers[BOTDATA any, USERDATA any] struct {
	TextHandler     func(*Session[BOTDATA, USERDATA], string, *Message)
	CommandHandlers map[string]func(*Session[BOTDATA, USERDATA], string, *Message) CmdResult
}

type Client[BOTDATA any, USERDATA any] struct {
	bot         *botImpl
	Firebase    Firebase[BOTDATA, USERDATA]
	Preference  Preference[BOTDATA]
	Sessions    map[int64]*Session[BOTDATA, USERDATA]
	Handlers    Handlers[BOTDATA, USERDATA]
	mu          sync.RWMutex
	delegate    ClientDelegate[BOTDATA, USERDATA]
	globalQueue *DispatchQueue
}

func (c *Client[BOTDATA, USERDATA]) Bot() BotAPI {
	return c.bot
}

func newClient[BOTDATA any, USERDATA any](config Config, delegate ClientDelegate[BOTDATA, USERDATA]) (*Client[BOTDATA, USERDATA], error) {
	client := &Client[BOTDATA, USERDATA]{
		Sessions: make(map[int64]*Session[BOTDATA, USERDATA]),
		Handlers: Handlers[BOTDATA, USERDATA]{
			CommandHandlers: make(map[string]func(*Session[BOTDATA, USERDATA], string, *Message) CmdResult),
		},
		delegate:    delegate,
		globalQueue: NewDispatchQueue(1, 100),
	}

	client.globalQueue.SetProcessHandler(client.processUpdate)

	if err := client.initBot(config.TelegramBotToken); err != nil {
		return nil, err
	}

	if err := client.initFirebase(config.FirebaseCredential, config.FirebaseDatabaseURL); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client[BOTDATA, USERDATA]) initFirebase(credential []byte, databaseURL string) error {
	ctx := context.Background()
	opt := option.WithCredentialsJSON(credential)
	conf := &firebase.Config{
		DatabaseURL: databaseURL,
	}
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return err
	}

	firestore, err := app.Firestore(ctx)
	if err != nil {
		return err
	}

	database, err := app.Database(ctx)
	if err != nil {
		return err
	}

	c.Firebase = Firebase[BOTDATA, USERDATA]{
		Firestore: firestore,
		Database:  database,
		Context:   ctx,
	}

	return nil
}

func (c *Client[BOTDATA, USERDATA]) initBot(token string) error {
	bi, err := newBotImpl(token, func(u *Update) {
		c.globalQueue.Enqueue(u)
	})
	if err != nil {
		return err
	}
	c.bot = bi
	return nil
}

func (c *Client[BOTDATA, USERDATA]) start() error {
	users, err := c.Firebase.GetUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		session := newSession(user, c)
		c.insertSession(session)

		c.delegate.DidLoadUser(session, user)
	}

	c.reload()

	c.globalQueue.Start()
	ctx, cancel := context.WithCancel(context.Background())
	c.bot.setCancel(cancel)
	go c.bot.Start(ctx)
	return nil
}

func (c *Client[BOTDATA, USERDATA]) stop() {
	c.bot.Stop()
	c.globalQueue.Stop()
}

func (c *Client[BOTDATA, USERDATA]) reload() error {
	var preference Preference[BOTDATA]
	if err := c.Firebase.Database.NewRef("/preference").Get(c.Firebase.Context, &preference); err != nil {
		return err
	}
	for key, val := range preference.Texts.Localizations {
		if bytes, err := base64.StdEncoding.DecodeString(val); err == nil {
			preference.Texts.Localizations[key] = string(bytes)
		}
	}
	for key, val := range preference.Texts.Prompts {
		if bytes, err := base64.StdEncoding.DecodeString(val); err == nil {
			preference.Texts.Prompts[key] = string(bytes)
		}
	}
	c.Preference = preference
	c.delegate.DidLoadPreference()
	return nil
}

func (c *Client[BOTDATA, USERDATA]) registerTextHandler(handler func(*Session[BOTDATA, USERDATA], string, *Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.TextHandler = handler
}

func (c *Client[BOTDATA, USERDATA]) registerCommandHandler(cmd string, handler func(*Session[BOTDATA, USERDATA], string, *Message) CmdResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Handlers.CommandHandlers[cmd] = handler
}

func (c *Client[BOTDATA, USERDATA]) getSession(id int64) *Session[BOTDATA, USERDATA] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Sessions[id]
}

func (c *Client[BOTDATA, USERDATA]) insertSession(session *Session[BOTDATA, USERDATA]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sessions[session.ID] = session
}

func (c *Client[BOTDATA, USERDATA]) processUpdate(update *Update) {
	if update == nil || update.Message == nil {
		return
	}

	message := update.Message
	id := message.Chat.ID
	session := c.getSession(id)
	if session == nil {
		user := &User[USERDATA]{
			ID:       id,
			UserData: c.delegate.NewUserData(),
		}
		if err := c.Firebase.UpdateUser(user); err != nil {
			return
		}

		session = newSession(user, c)
		c.insertSession(session)

		c.delegate.DidLoadUser(session, user)
	}

	c.processMessage(session, message)
}

func (c *Client[BOTDATA, USERDATA]) processMessage(session *Session[BOTDATA, USERDATA], message *Message) {
	if session.User.Blocked {
		session.User.Blocked = false
		c.Firebase.UpdateUser(session.User)
	}

	if message.IsCommand() {
		c.processCommand(session, message.Command(), message.CommandArguments(), message)
	} else if session.CommandSession.Command != "" {
		c.processCommand(session, session.CommandSession.Command, message.Text, message)
	} else {
		c.processText(session, message.Text, message)
	}
}

func (c *Client[BOTDATA, USERDATA]) processCommand(session *Session[BOTDATA, USERDATA], command string, args string, message *Message) {
	switch command {
	case CmdStart:
		session.SendTextWithConfig("Greetings.", MessageConfig{
			PromptKey:        CmdStart,
			ReplyToMessageID: message.MessageID,
		})
		return
	case CmdBotReload:
		if _, rv := c.Preference.Admins[message.Chat.ID]; rv {
			c.reload()
			session.ReplyText("Done.", message.MessageID)
		}
		return
	case CmdBotStat:
		if _, rv := c.Preference.Admins[message.Chat.ID]; rv {
			session.ReplyText(fmt.Sprintf("Total Users: %d", len(c.Sessions)), message.MessageID)
		}
		return
	}

	if c.Preference.OnlyAdminsCanCommandInGroup && (message.Chat.IsSuperGroup() || message.Chat.IsGroup()) {
		adminIDs, err := c.bot.GetChatAdministratorIDs(context.Background(), message.Chat.ID)
		if err != nil {
			return
		}

		isAdmin := false
		fromID := int64(0)
		if message.From != nil {
			fromID = message.From.ID
		}
		for _, id := range adminIDs {
			if id == fromID {
				isAdmin = true
				break
			}
		}
		if !isAdmin && fromID == int64(GroupAnonymousBot) {
			isAdmin = true
		}
		if !isAdmin {
			return
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	handler, exists := c.Handlers.CommandHandlers[command]
	if !exists {
		return
	}

	if command != session.CommandSession.Command {
		session.CommandSession.Command = command
		session.CommandSession.Stage = ""
		session.CommandSession.Args = make(map[string]any)
	}
	result := handler(session, args, message)
	if result == CmdResultProcessed {
		session.CommandSession.Command = ""
		session.CommandSession.Stage = ""
		session.CommandSession.Args = make(map[string]any)
	}
}

func (c *Client[BOTDATA, USERDATA]) processText(session *Session[BOTDATA, USERDATA], text string, message *Message) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler := c.Handlers.TextHandler
	if handler != nil {
		handler(session, text, message)
	}
}

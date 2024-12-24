package tgbot

import (
	"context"
	"log"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"google.golang.org/api/option"
)

type Client struct {
	BotAPI          *tgbotapi.BotAPI
	Firebase        Firebase
	CCMS            CCMS
	Sessions        map[int64]*Session
	CommandHandlers map[string]func(Session, string) bool
	TextHandler     func(Session, string) bool
	mu              sync.RWMutex
}

func (c *Client) start(token string, credential []byte, databaseURL string) error {
	if err := c.startFirebase(credential, databaseURL); err != nil {
		return err
	}

	if err := c.reloadCCMS(); err != nil {
		return err
	}

	if err := c.startBotAPI(token); err != nil {
		return err
	}

	users, err := c.Firebase.getUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		session := &Session{
			ID:     user.ID,
			User:   user,
			client: c,
		}
		c.insertSession(session)
	}

	go c.runLoop()

	return nil
}

func (c *Client) startFirebase(credential []byte, databaseURL string) error {
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

	firebase := Firebase{
		Firestore: firestore,
		Database:  database,
		Context:   context,
	}
	c.Firebase = firebase

	return nil
}

func (c *Client) startBotAPI(token string) error {
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

func (c *Client) reloadCCMS() error {
	var CCMS CCMS
	if err := c.Firebase.Database.NewRef("/ccms").Get(c.Firebase.Context, &CCMS); err != nil {
		return err
	}
	c.CCMS = CCMS
	return nil
}

func (c *Client) registerCommandHandler(cmd string, handler func(Session, string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CommandHandlers[cmd] = handler
}

func (c *Client) registerTextHandler(handler func(Session, string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TextHandler = handler
}

func (c *Client) runLoop() {
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

func (c *Client) getSession(id int64) *Session {
	return c.Sessions[id]
}

func (c *Client) insertSession(session *Session) {
	c.Sessions[session.ID] = session
}

func (c *Client) processUpdate(update tgbotapi.Update) {
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
		user := &User{
			ID: id,
		}
		if err := c.Firebase.updateUser(user); err != nil {
			return
		}

		session = &Session{
			ID:     id,
			User:   user,
			client: c,
		}
		c.insertSession(session)
	}

	c.processMessage(session, message)
}

func (c *Client) processMessage(session *Session, message *tgbotapi.Message) {
	if session.User.Blocked {
		session.User.Blocked = false
		c.Firebase.updateUser(session.User)
	}

	if message.IsCommand() {
		switch message.Command() {
		case CmdStart:
			session.SendFormattedText(session, "Greetings.", CmdStart)

		case CmdReload:
			if _, rv := c.CCMS.Admins[message.Chat.ID]; rv {
				c.reloadCCMS()
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

func (c *Client) processCommand(session *Session, command string, args string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler, exists := c.CommandHandlers[command]
	if !exists {
		return
	}

	if handler(*session, args) {
		session.command = ""
	} else {
		session.command = command
	}
}

func (c *Client) processText(session *Session, text string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handler := c.TextHandler
	if handler != nil {
		handler(*session, text)
	}
}

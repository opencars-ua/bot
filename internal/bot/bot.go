package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shal/robot/pkg/env"
	"github.com/shal/robot/pkg/openalpr"
	"github.com/shal/robot/pkg/opencars"
)

// Handler ...
type Handler interface {
	Handle(bot *tgbotapi.BotAPI, msg *tgbotapi.Message)
}

// HandlerFunc ...
type HandlerFunc func(*tgbotapi.BotAPI, *tgbotapi.Message)

// HandlerFunc ...
func (f HandlerFunc) Handle(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	f(bot, msg)
}

// MuxEntry ...
type MuxEntry struct {
	handler Handler
	match   func(x string) bool
}

// Bot ...
type Bot struct {
	API        *tgbotapi.BotAPI
	Recognizer *openalpr.API
	Storage    *opencars.API
	FilePath   string
	Mux        []MuxEntry
}

// Handle ...
func (bot *Bot) Handle(key string, handler Handler) {
	bot.Mux = append(bot.Mux, MuxEntry{
		handler: handler,
		match: func(text string) bool {
			return strings.HasPrefix(text, key)
		},
	})
}

// HandleRegexp ...
func (bot *Bot) HandleRegexp(regexp *regexp.Regexp, handler Handler) {
	bot.Mux = append(bot.Mux, MuxEntry{
		handler: handler,
		match: func(x string) bool {
			return regexp.MatchString(x)
		},
	})
}

// HandleFuncRegexp ...
func (bot *Bot) HandleFuncRegexp(regexp *regexp.Regexp, handler func(*tgbotapi.BotAPI, *tgbotapi.Message)) {
	bot.Mux = append(bot.Mux, MuxEntry{
		handler: HandlerFunc(handler),
		match: func(x string) bool {
			return regexp.MatchString(x)
		},
	})
}

// HandleFunc ...
func (bot *Bot) HandleFunc(key string, handler func(*tgbotapi.BotAPI, *tgbotapi.Message)) {
	bot.Mux = append(bot.Mux, MuxEntry{
		handler: HandlerFunc(handler),
		match: func(text string) bool {
			return strings.HasPrefix(text, key)
		},
	})
}

func (bot *Bot) handle(update tgbotapi.Update) {
	for _, entry := range bot.Mux {
		if update.Message == nil {
			break
		}

		if entry.match(update.Message.Text) {
			entry.handler.Handle(bot.API, update.Message)
		}
	}
}

// Listen
func (bot *Bot) Listen(host, port string) error {
	URL := fmt.Sprintf("%s/%s", host, bot.API.Token)
	_, err := bot.API.SetWebhook(tgbotapi.NewWebhook(URL))
	if err != nil {
		log.Fatal(err)
	}

	// Should be thread safe out of the box.
	path := fmt.Sprintf("/%s", bot.API.Token)

	log.Println(path)

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()

		update := tgbotapi.Update{}

		if err := json.Unmarshal(bytes, &update); err != nil {
			log.Printf("update error: %s", err.Error())
		}

		fmt.Printf("Incoming request %v\n", r)
		// Handle "Update".
		bot.handle(update)
	})

	return http.ListenAndServe(":"+port, http.DefaultServeMux)
}

func New(path, recognizerUrl, storageUrl string) *Bot {
	return &Bot{
		API:        NewAPI(),
		Recognizer: &openalpr.API{URL: recognizerUrl},
		Storage:    &opencars.API{URI: storageUrl},
		Mux:        make([]MuxEntry, 0),
		FilePath:   path,
	}
}

func NewAPI() *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(env.MustGet("BOT_TOKEN"))

	bot.Debug = env.Get("LOG_LEVEL", "DEBUG") == "DEBUG"

	if err != nil {
		panic(err)
	}

	log.Printf("API authorized %s\n", bot.Self.UserName)

	return bot
}

func send(bot *tgbotapi.BotAPI, message tgbotapi.MessageConfig) error {
	if _, err := bot.Send(message); err != nil {
		return err
	}

	return nil
}

// SendHTML send message to the chat with regular text.
func Send(bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, text string) error {
	msg := tgbotapi.NewMessage(chat.ID, text)
	return send(bot, msg)
}

// SendHTML send message to the chat with text formatted as HTML.
func SendHTML(bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, text string) error {
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = tgbotapi.ModeHTML

	return send(bot, msg)
}

// SendHTML send message to the chat with text formatted as HTML.
func SendMsgHTML(msg tgbotapi.MessageConfig, bot *tgbotapi.BotAPI) error {
	msg.ParseMode = tgbotapi.ModeHTML

	return send(bot, msg)
}

// SendMarkdown send message to the chat with text formatted as Markdown.
func SendMarkdown(bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, text string) error {
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	return send(bot, msg)
}
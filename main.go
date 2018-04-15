package main

import (
	"context"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strings"
	"time"
	"strconv"
)

const (
	MaxQueryLen                 = 150
	LimiterOpName               = "request"
	LimiterDefaultTimeoutMillis = 10000
	UnknownError                = "Something went wrong."
	InputTooLong                = "Your request is too long."
	NoInputProvided		    = "Please provide your query in plaintext."
	TelegramKeyEnv              = "TELEGRAM_KEY_CODING_GURU_BOT"
	DevModeEnv 		    = "DEV_MODE_CODING_GURU_BOT"
	HelpString                  = `Just type your query and send!
Examples:
- c++ check if element in list
- schedule task java
- binary search go
- docker syncSet env
`
)

func main() {
	telegramKey := os.Getenv(TelegramKeyEnv)
	if len(telegramKey) == 0 {
		log.Fatalf("[FATAL] No Telegram Key provided. Please syncSet %s environment variable.", TelegramKeyEnv)
	}
	bot, err := tgbotapi.NewBotAPI(telegramKey)
	if err != nil {
		log.Fatalf("[FATAL] Failed to register: %v", err)
	}
	log.Printf("[INFO] Bot authorized - %s", bot.Self.UserName)
	var updatesChan tgbotapi.UpdatesChannel
	if len(os.Getenv(DevModeEnv)) != 0 {
		updatesChan = pollingChannel(bot)
	} else {
		updatesChan = webHookChannel(bot)
	}
	log.Printf("[INFO] Updates channel obtained")
	processUpdates(bot, updatesChan)
}

func pollingChannel(bot *tgbotapi.BotAPI) tgbotapi.UpdatesChannel {
	log.Printf("[INFO] Registering a polling-based updates channel...")
	updateConfig := tgbotapi.NewUpdate(0)
	channel, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Fatalf("[FATAL] Failed to get polling updates channel: %v", err)
	}
	return channel
}

func webHookChannel(bot *tgbotapi.BotAPI) tgbotapi.UpdatesChannel {
	panic("implement me")
}

func processUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	guru := NewGuru()
	limiter := NewLimiter()
	for update := range updates {
		go func() {
			if update.Message == nil {
				return
			}
			username := update.Message.From.UserName
			userId := strconv.Itoa(update.Message.From.ID)
			chatId := update.Message.Chat.ID

			allowed, dropLimit := limiter.Allow(LimiterOpName, userId, LimiterDefaultTimeoutMillis)
			if !allowed {
				log.Printf("[WARN] Request from user '%s'(id=%s) already in progress.", username, userId)
				return
			}
			defer dropLimit()
			if update.Message.IsCommand() {
				command := update.Message.Command()
				log.Printf("[INFO] Received command (from = %s, command = %s)", username, command)
				switch command {
				case "help":
					respond(bot, chatId, HelpString)
				default:
					log.Printf("[WARN] Unknown command - %s", command)
				}
				return
			}

			query := strings.TrimSpace(strings.ToLower(update.Message.Text))
			log.Printf("[INFO] Received query (from = %s, query = %s)", username, query)
			inputLen := len(query)
			if inputLen == 0 {
				log.Printf("[ERROR] Empty query (username = %v)", username)
				respond(bot, chatId, NoInputProvided)
				return
			}
			if inputLen > MaxQueryLen {
				log.Printf("[ERROR] Query is too long (username = %v, max = %v, actual = %v)",
					username, MaxQueryLen, inputLen)
				respond(bot, chatId, InputTooLong)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3*time.Second))
			defer cancel()
			result, err := guru.FindAnswer(ctx, query)
			if err != nil {
				log.Printf("[ERROR] Failed to run query: %v", err)
				respond(bot, chatId, UnknownError)
				return
			}
			respond(bot, chatId, result)
		}()
	}
}

func respond(bot *tgbotapi.BotAPI, chat int64, message string) error {
	response := tgbotapi.NewMessage(chat, message)
	_, err := bot.Send(response)
	if err != nil {
		return err
	}
	return nil
}

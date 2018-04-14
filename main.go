package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strings"
	"os"
	"context"
	"time"
)

const (
	MaxQueryLen    = 150
	UnknownError   = "Something went wrong."
	InputTooLong   = "Your request is too long."
	TelegramKeyEnv = "TELEGRAM_KEY_CODING_GURU_BOT"
	HelpString     = `Just type your query and send!
Examples:
- c++ check if element in list
- schedule task java
- binary search go
- docker set env
`
)

func main() {
	telegramKey := os.Getenv(TelegramKeyEnv)
	if len(telegramKey) == 0 {
		log.Fatalf("[FATAL] No Telegram Key provided. Please set %s environment variable.", TelegramKeyEnv)
	}
	bot, err := tgbotapi.NewBotAPI(telegramKey)
	if err != nil {
		log.Fatalf("[FATAL] Failed to register: %v", err)
	}
	log.Printf("[INFO] Bot authorized - %s", bot.Self.UserName)
	updateConfig := tgbotapi.NewUpdate(0)
	updatesChan, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Fatalf("[FATAL] Failed to get updates channel: %v", err)
	}
	var guru CodeGuru = &WebCrawlerCodeGuru{}
	for update := range updatesChan {
		go func() {
			if update.Message == nil {
				return
			}
			username := update.Message.From.UserName // TODO throttle based on name/id
			chatId := update.Message.Chat.ID

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
				respond(bot, chatId, InputTooLong)
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
			result, err := guru.findAnswer(ctx, query)
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

package main

import (
	"context"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	MaxQueryLen                 = 150
	LimiterDefaultTimeoutMillis = 10000
	LimiterOpName               = "request"
	UnknownError                = "Something went wrong."
	InputTooLong                = "Your request is too long."
	NoInputProvided		    = "Please provide your query in plaintext."
	TelegramKeyEnv              = "TELEGRAM_KEY_CODING_GURU_BOT"
	DevModeEnv                  = "DEV_MODE_CODING_GURU_BOT"
	WebHookHostEnv              = "WEB_HOOK_HOST_CODING_GURU_BOT"
	WebHookListenPortEnv        = "WEB_HOOK_LISTEN_PORT_CODING_GURU_BOT"
	UseTLSEnv                   = "USE_TLS_ENCRYPTION_CODING_GURU_BOT"
	TLSCertFileEnv              = "TLS_CERT_FILE_CODING_GURU_BOT"
	TLSKeyFileEnv               = "TLS_KEY_FILE_CODING_GURU_BOT"
	HelpString                  = `Just type your query and send!
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
	log.Printf("[INFO] Registering a webhook-based updates channel...")
	link := os.Getenv(WebHookHostEnv)
	if len(link) == 0 {
		log.Fatalf("[FATAL] No host for webhook provided. Please set %s environment variable.", WebHookHostEnv)
	}
	port := os.Getenv(WebHookListenPortEnv)
	if len(link) == 0 {
		log.Fatalf("[FATAL] No port for webhook provided. Please set %s environment variable.", WebHookListenPortEnv)
	}

	address := fmt.Sprintf("0.0.0.0:%s", port)
	var webHookConfig tgbotapi.WebhookConfig
	var httpServeFunc func()
	useTls := len(os.Getenv(UseTLSEnv)) > 0
	if useTls {
		certFile := os.Getenv(TLSCertFileEnv)
		if len(certFile) == 0 {
			log.Fatalf("[FATAL] No cert file provided. Please set %s environment variable.", TLSCertFileEnv)
		}
		keyFile := os.Getenv(TLSKeyFileEnv)
		if len(keyFile) == 0 {
			log.Fatalf("[FATAL] No cert key file provided. Please set %s environment variable.", TLSKeyFileEnv)
		}
		webHookConfig = tgbotapi.NewWebhookWithCert(link+bot.Token, certFile)
		httpServeFunc = func() {
			go http.ListenAndServeTLS(address, certFile, keyFile, nil)
		}
	} else {
		webHookConfig = tgbotapi.NewWebhook(link + bot.Token)
		httpServeFunc = func() {
			go http.ListenAndServe(address, nil)
		}
	}
	_, err := bot.SetWebhook(webHookConfig)
	if err != nil {
		log.Fatalf("[FATAL] Could not register web hook - %v", err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatalf("[FATAL] Could not get web hook info - %v", err)
	}
	if info.LastErrorDate != 0 {
		log.Fatalf("[FATAL] Web hook callback failed - %v", info.LastErrorMessage)
	}
	channel := bot.ListenForWebhook("/" + bot.Token)
	httpServeFunc()
	return channel
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

package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is empty. Set env var, e.g. export TELEGRAM_BOT_TOKEN=...")
	}

	mainBotUsername := "volgogradVPN_bot"
	mainBotURL := "https://t.me/" + mainBotUsername

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	startText := " 🖐️Привет! Высокоскоростное подключение к любым сайтам и бесперебойная работа всего в 1 шаге от тебя!\n\n" +
		"Запускай основного бота ниже и пользуйся сервисом 5 ДНЕЙ на 3 УСТРОЙСТВАХ без ограничений в скорости и качестве!🤩"

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			btn := tgbotapi.NewInlineKeyboardButtonURL("🔥Запустить основного бота", mainBotURL)
			kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, startText)
			msg.ReplyMarkup = kb

			if _, err := bot.Send(msg); err != nil {
				log.Println("send error:", err)
			}
		}
	}
}

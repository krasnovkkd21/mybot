package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is empty. Set env var TELEGRAM_BOT_TOKEN=...")
	}

	// Основной бот — кнопка ведет сюда
	mainBotUsername := "volgogradVPN_bot"
	mainBotURL := "https://t.me/" + mainBotUsername

	// SQLite база в файле events.db рядом с бинарником
	db, err := sql.Open("sqlite3", "./events.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

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
			// /start <param>
			kw := strings.TrimSpace(update.Message.CommandArguments())
			if kw == "" {
				kw = "organic"
			}

			user := update.Message.From
			chatID := update.Message.Chat.ID

			// 1) сохраняем/обновляем юзера
			if err := upsertUser(db, user); err != nil {
				log.Println("db upsertUser error:", err)
			}

			// 2) логируем старт (для аналитики)
			if err := logStart(db, kw, user.ID, chatID); err != nil {
				log.Println("db logStart error:", err)
			}

			// 3) показываем текст + URL-кнопку (переход в основной бот)
			btn := tgbotapi.NewInlineKeyboardButtonURL("🔥Запустить основного бота", mainBotURL)
			kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))

			msg := tgbotapi.NewMessage(chatID, startText)
			msg.ReplyMarkup = kb

			if _, err := bot.Send(msg); err != nil {
				log.Println("send error:", err)
			}
		}
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  user_id INTEGER PRIMARY KEY,
  username TEXT,
  first_name TEXT,
  last_name TEXT,
  updated_ts TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS starts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  kw TEXT NOT NULL,
  user_id INTEGER NOT NULL,
  chat_id INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_starts_kw ON starts(kw);
CREATE INDEX IF NOT EXISTS idx_starts_user ON starts(user_id);
`)
	return err
}

func upsertUser(db *sql.DB, u *tgbotapi.User) error {
	_, err := db.Exec(`
INSERT INTO users (user_id, username, first_name, last_name, updated_ts)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
  username=excluded.username,
  first_name=excluded.first_name,
  last_name=excluded.last_name,
  updated_ts=excluded.updated_ts;
`,
		u.ID, u.UserName, u.FirstName, u.LastName,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func logStart(db *sql.DB, kw string, userID int64, chatID int64) error {
	_, err := db.Exec(`
INSERT INTO starts (ts, kw, user_id, chat_id)
VALUES (?, ?, ?, ?);
`,
		time.Now().Format(time.RFC3339),
		kw,
		userID,
		chatID,
	)
	return err
}

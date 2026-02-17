package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is empty")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is empty")
	}

	// Куда ведет кнопка
	mainBotURL := "https://t.me/volgogradVPN_bot"

	// Текст /start
	startText := "🖐️Привет! Высокоскоростное подключение к любым сайтам и бесперебойная работа всего в 1 шаге от тебя!\n\n" +
		"Запускай основного бота ниже и пользуйся сервисом 5 ДНЕЙ на 3 УСТРОЙСТВАХ без ограничений в скорости и качестве!🤩"

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("pgxpool.New error:", err)
	}
	defer pool.Close()

	// Создаем таблицы, если их нет
	if err := initDB(ctx, pool); err != nil {
		log.Fatal("initDB error:", err)
	}

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			kw := strings.TrimSpace(update.Message.CommandArguments())
			if kw == "" {
				kw = "organic"
			}

			user := update.Message.From
			chatID := update.Message.Chat.ID

			// upsert user
			if err := upsertUser(ctx, pool, user); err != nil {
				log.Println("upsertUser error:", err)
			}

			// log start
			if err := logStart(ctx, pool, kw, user.ID, chatID); err != nil {
				log.Println("logStart error:", err)
			}

			// URL-кнопка (переход в основного бота)
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

func initDB(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := `
CREATE TABLE IF NOT EXISTS users (
  user_id BIGINT PRIMARY KEY,
  username TEXT,
  first_name TEXT,
  last_name TEXT,
  updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS starts (
  id BIGSERIAL PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  kw TEXT NOT NULL,
  user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  chat_id BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_starts_kw ON starts(kw);
CREATE INDEX IF NOT EXISTS idx_starts_user ON starts(user_id);
`
	_, err := pool.Exec(ctx, ddl)
	return err
}

func upsertUser(ctx context.Context, pool *pgxpool.Pool, u *tgbotapi.User) error {
	_, err := pool.Exec(ctx, `
INSERT INTO users (user_id, username, first_name, last_name, updated_ts)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
  username = EXCLUDED.username,
  first_name = EXCLUDED.first_name,
  last_name = EXCLUDED.last_name,
  updated_ts = EXCLUDED.updated_ts
`,
		u.ID,
		nullIfEmpty(u.UserName),
		nullIfEmpty(u.FirstName),
		nullIfEmpty(u.LastName),
		time.Now(),
	)
	return err
}

func logStart(ctx context.Context, pool *pgxpool.Pool, kw string, userID int64, chatID int64) error {
	_, err := pool.Exec(ctx, `
INSERT INTO starts (kw, user_id, chat_id)
VALUES ($1, $2, $3)
`,
		kw, userID, chatID,
	)
	return err
}

// helper: чтобы не писать пустые строки (можно и без него, но так аккуратнее)
func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

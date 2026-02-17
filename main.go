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

const (
	cbOpenMainBot = "open_main_bot"
)

var allowedParams = map[string]bool{
	"kw_vpsru":        true,
	"kw_vps":          true,
	"kw_freevps":      true,
	"kw_vpsforpc":     true,
	"kw_happvps":      true,
	"kw_freevpsen":    true,
	"kw_vpsandroid":   true,
	"kw_vpsiphone":    true,
	"kw_vpswhitelist": true,
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is empty")
	}

	// username основного бота (без @)
	mainBotUsername := os.Getenv("MAIN_BOT_USERNAME")
	if mainBotUsername == "" {
		mainBotUsername = "MainBotUsernameHere"
	}
	mainBotURL := "https://t.me/" + mainBotUsername

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

	startText := "🖐️Привет! Высокоскоростное подключение к любым сайтам и бесперебойная работа всего в 1 шаге от тебя!\n\n" +
		"Запускай основного бота ниже и пользуйся сервисом 5 ДНЕЙ на 3 УСТРОЙСТВАХ без ограничений в скорости и качестве!🤩"

	for update := range updates {

		// /start <param>
		if update.Message != nil && update.Message.IsCommand() && update.Message.Command() == "start" {
			param := strings.TrimSpace(update.Message.CommandArguments())
			if param == "" {
				param = "organic"
			}
			// если хочешь учитывать только твои kw_* — раскомментируй:
			// if param != "organic" && !allowedParams[param] { param = "other" }

			usr := update.Message.From
			_ = upsertUserFirstSeen(db, usr)

			if err := logEvent(db, "start", param, usr, update.Message.Chat.ID); err != nil {
				log.Println("db start log error:", err)
			}

			btn := tgbotapi.NewInlineKeyboardButtonData("🔥Запустить основного бота", cbOpenMainBot)
			kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, startText)
			msg.ReplyMarkup = kb
			if _, err := bot.Send(msg); err != nil {
				log.Println("send error:", err)
			}
			continue
		}

		// callback клик по кнопке
		if update.CallbackQuery != nil {
			q := update.CallbackQuery
			usr := q.From
			_ = upsertUserFirstSeen(db, usr)

			if q.Data == cbOpenMainBot {
				// привяжем к последнему start_param пользователя
				lastParam := getLastStartParam(db, usr.ID)

				if err := logEvent(db, "click_main_bot", lastParam, usr, q.Message.Chat.ID); err != nil {
					log.Println("db click log error:", err)
				}

				_, _ = bot.Request(tgbotapi.NewCallback(q.ID, "Готово ✅"))

				out := tgbotapi.NewMessage(q.Message.Chat.ID,
					"Вот основной бот: "+mainBotURL+"\n\nНажми и запусти его.")
				if _, err := bot.Send(out); err != nil {
					log.Println("send main bot url error:", err)
				}
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
	first_seen_ts TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts TEXT NOT NULL,
	event_type TEXT NOT NULL,     -- start / click_main_bot
	start_param TEXT NOT NULL,    -- kw_* / organic / other
	user_id INTEGER NOT NULL,
	chat_id INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_user ON events(user_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_param ON events(start_param);
`)
	return err
}

func upsertUserFirstSeen(db *sql.DB, usr *tgbotapi.User) error {
	// сохраняем first_seen один раз, но обновляем username/name (на случай если менялись)
	_, err := db.Exec(`
INSERT INTO users (user_id, username, first_name, last_name, first_seen_ts)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
	username=excluded.username,
	first_name=excluded.first_name,
	last_name=excluded.last_name
`,
		usr.ID, usr.UserName, usr.FirstName, usr.LastName,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func logEvent(db *sql.DB, eventType, startParam string, usr *tgbotapi.User, chatID int64) error {
	_, err := db.Exec(`
INSERT INTO events (ts, event_type, start_param, user_id, chat_id)
VALUES (?, ?, ?, ?, ?)
`,
		time.Now().Format(time.RFC3339),
		eventType,
		startParam,
		usr.ID,
		chatID,
	)
	return err
}

func getLastStartParam(db *sql.DB, userID int64) string {
	var sp string
	err := db.QueryRow(`
SELECT start_param FROM events
WHERE user_id = ? AND event_type = 'start'
ORDER BY id DESC LIMIT 1
`, userID).Scan(&sp)
	if err != nil || sp == "" {
		return "unknown"
	}
	return sp
}

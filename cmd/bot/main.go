package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/yourusername/2gis-parser/internal/config"
	"github.com/yourusername/2gis-parser/internal/models"
	"github.com/yourusername/2gis-parser/internal/parser"
	"github.com/yourusername/2gis-parser/internal/storage"
)

// userSession stores a user's dialog state.
type userSession struct {
	State  string
	Query  string
	Cities []string
}

var (
	sessions   = make(map[int64]*userSession)
	sessionsMu sync.Mutex
)

var cityDisplayNames = map[string]string{
	"Astana":        "Астана",
	"Almaty":        "Алматы",
	"Kostanay":      "Костанай",
	"Kokshetau":     "Кокшетау",
	"Karaganda":     "Караганда",
	"Petropavlovsk": "Петропавловск",
}

func main() {
	cfg := config.Load()

	// Create directories.
	for _, dir := range []string{"./data", cfg.OutputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	db, err := storage.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	log.Printf("🤖 Bot started: @%s", bot.Self.UserName)

	p := parser.New(cfg.DGISAPIKey, cfg.MaxPages)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			go handleCallback(bot, update.CallbackQuery, p, db, cfg)
			continue
		}
		if update.Message == nil {
			continue
		}
		go handleMessage(bot, update.Message, p, db, cfg)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, p *parser.Parser, db *storage.DB, cfg *config.Config) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)

	sessionsMu.Lock()
	sess, exists := sessions[userID]
	if !exists {
		sess = &userSession{State: "idle"}
		sessions[userID] = sess
	}
	sessionsMu.Unlock()

	switch {
	case text == "/start":
		sendWelcome(bot, msg.Chat.ID)

	case text == "/help":
		sendHelp(bot, msg.Chat.ID)

	case text == "/parse" || text == "🔍 Start Parsing" || text == "🔍 Начать парсинг":
		sess.State = "awaiting_query"
		send(bot, msg.Chat.ID, "📝 *Введите поисковый запрос*\n\nПримеры: `кассовая лента`, `кассовые аппараты`, `POS-терминалы`", nil)

	case text == "/status":
		send(bot, msg.Chat.ID, "ℹ️ Бот работает нормально. Используйте /parse, чтобы начать парсинг.", nil)

	case sess.State == "awaiting_query":
		sess.Query = text
		sess.State = "awaiting_cities"
		sendCityPicker(bot, msg.Chat.ID)

	default:
		sendWelcome(bot, msg.Chat.ID)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, p *parser.Parser, db *storage.DB, cfg *config.Config) {
	userID := cb.From.ID
	data := cb.Data

	sessionsMu.Lock()
	sess, exists := sessions[userID]
	if !exists {
		sess = &userSession{State: "idle"}
		sessions[userID] = sess
	}
	sessionsMu.Unlock()

	if data == "parse_start" {
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		sess.State = "awaiting_query"
		sess.Query = ""
		sess.Cities = nil
		send(bot, cb.Message.Chat.ID, "📝 *Введите поисковый запрос*\n\nПримеры: `кассовая лента`, `кассовые аппараты`, `POS-терминалы`", nil)
		return
	}

	if sess.State != "awaiting_cities" {
		bot.Request(tgbotapi.NewCallback(cb.ID, "Сначала начните новый парсинг и введите запрос."))
		return
	}

	switch {
	case data == "parse_all":
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		sess.Cities = []string{"Astana", "Almaty", "Kostanay", "Kokshetau", "Karaganda", "Petropavlovsk"}
		sess.State = "parsing"
		startParsing(bot, cb.Message.Chat.ID, sess, p, db, cfg)

	case strings.HasPrefix(data, "city_"):
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		cityName := strings.TrimPrefix(data, "city_")
		// Toggle the city.
		found := false
		for i, c := range sess.Cities {
			if c == cityName {
				sess.Cities = append(sess.Cities[:i], sess.Cities[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			sess.Cities = append(sess.Cities, cityName)
		}
		// Update the message.
		editCityPicker(bot, cb.Message.Chat.ID, cb.Message.MessageID, sess.Cities)

	case data == "start_parse":
		if len(sess.Cities) == 0 {
			bot.Request(tgbotapi.NewCallback(cb.ID, "⚠️ Выберите хотя бы один город!"))
			return
		}
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		sess.State = "parsing"
		startParsing(bot, cb.Message.Chat.ID, sess, p, db, cfg)

	default:
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	}
}

func startParsing(bot *tgbotapi.BotAPI, chatID int64, sess *userSession, p *parser.Parser, db *storage.DB, cfg *config.Config) {
	query := sess.Query
	cities := append([]string(nil), sess.Cities...)
	defer func() {
		sessionsMu.Lock()
		*sess = userSession{State: "idle"}
		sessionsMu.Unlock()
	}()

	status := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("⏳ *Запускаю парсинг...*\n\n🔍 Запрос: `%s`\n🏙 Города: %s\n\n_Пожалуйста, подождите_",
			query, formatCityList(cities)))
	status.ParseMode = "Markdown"
	statusMsg, _ := bot.Send(status)

	var allCompanies []models.Company
	results := make([]string, 0)

	for _, city := range cities {
		edit(bot, chatID, statusMsg.MessageID,
			fmt.Sprintf("⏳ *Идет парсинг...*\n\n🔍 `%s`\n🏙 Текущий город: *%s*\n📊 Найдено: %d", query, cityDisplayName(city), len(allCompanies)))

		companies, err := p.Search(query, city, nil)
		if err != nil {
			results = append(results, fmt.Sprintf("❌ %s: ошибка - %v", cityDisplayName(city), err))
			log.Printf("Parsing error for %s: %v", city, err)
			continue
		}

		if len(companies) > 0 {
			// Set the city.
			for i := range companies {
				companies[i].City = city
			}
			if err := db.SaveCompanies(companies, city, query); err != nil {
				log.Printf("Failed to save %s: %v", city, err)
			}
			allCompanies = append(allCompanies, companies...)
		}

		results = append(results, fmt.Sprintf("✅ %s: *%d* компаний", cityDisplayName(city), len(companies)))
		time.Sleep(500 * time.Millisecond)
	}

	if len(allCompanies) == 0 {
		edit(bot, chatID, statusMsg.MessageID,
			fmt.Sprintf("😔 *Ничего не найдено*\n\nЗапрос: `%s`\n\n%s", query, strings.Join(results, "\n")))
		return
	}

	// Export to Excel.
	edit(bot, chatID, statusMsg.MessageID, "📊 *Создаю Excel-файл...*")

	xlsxPath, err := storage.ExportToExcel(allCompanies, cfg.OutputDir, query)
	if err != nil {
		edit(bot, chatID, statusMsg.MessageID, fmt.Sprintf("❌ Не удалось создать файл: %v", err))
		return
	}

	// Summary message.
	summary := fmt.Sprintf("✅ *Парсинг завершен!*\n\n🔍 Запрос: `%s`\n📦 Всего: *%d* компаний\n\n%s",
		query, len(allCompanies), strings.Join(results, "\n"))
	edit(bot, chatID, statusMsg.MessageID, summary)

	// Send the file.
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(xlsxPath))
	doc.Caption = fmt.Sprintf("📋 Результаты: *%d* компаний по запросу \"%s\"", len(allCompanies), query)
	doc.ParseMode = "Markdown"
	bot.Send(doc)
}

// ---- UI helpers ----

func sendWelcome(bot *tgbotapi.BotAPI, chatID int64) {
	text := `🏢 *2GIS Parser - Казахстан*

Этот бот собирает контакты компаний из 2GIS по городам Казахстана.

📦 *Возможности:*
• поиск компаний по любому запросу
• парсинг нескольких городов сразу
• экспорт результатов в Excel с телефонами, адресами и сайтами
• сохранение истории в базе данных

Нажмите кнопку ниже, чтобы начать:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Начать парсинг", "parse_start"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func sendHelp(bot *tgbotapi.BotAPI, chatID int64) {
	text := `📖 *Помощь*

*/parse* - начать новый парсинг
*/status* - статус бота
*/help* - это справочное сообщение

*Пример использования:*
1. Введите /parse
2. Введите запрос: _кассовая лента_
3. Выберите города (Астана, Костанай, ...)
4. Получите Excel-файл с результатами`

	send(bot, chatID, text, nil)
}

func sendCityPicker(bot *tgbotapi.BotAPI, chatID int64) {
	keyboard := buildCityKeyboard([]string{})
	msg := tgbotapi.NewMessage(chatID, "🏙 *Выберите города для парсинга:*\n\n_(можно выбрать несколько)_")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func editCityPicker(bot *tgbotapi.BotAPI, chatID int64, msgID int, selected []string) {
	keyboard := buildCityKeyboard(selected)
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID,
		fmt.Sprintf("🏙 *Выберите города:*\n\n✅ Выбрано: %d\n_%s_",
			len(selected), formatCityList(selected)),
		keyboard)
	edit.ParseMode = "Markdown"
	bot.Send(edit)
}

func buildCityKeyboard(selected []string) tgbotapi.InlineKeyboardMarkup {
	cities := []struct{ name, slug string }{
		{"Астана", "Astana"},
		{"Алматы", "Almaty"},
		{"Костанай", "Kostanay"},
		{"Кокшетау", "Kokshetau"},
		{"Караганда", "Karaganda"},
		{"Петропавловск", "Petropavlovsk"},
	}

	selectedMap := make(map[string]bool)
	for _, s := range selected {
		selectedMap[s] = true
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(cities); i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+2 && j < len(cities); j++ {
			label := cities[j].name
			if selectedMap[cities[j].slug] {
				label = "✅ " + label
			}
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, "city_"+cities[j].slug))
		}
		rows = append(rows, row)
	}

	// Action buttons.
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌐 Все города", "parse_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Начать парсинг", "start_parse"),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func cityDisplayName(city string) string {
	if name, ok := cityDisplayNames[city]; ok {
		return name
	}
	return city
}

func formatCityList(cities []string) string {
	if len(cities) == 0 {
		return "нет"
	}
	names := make([]string, 0, len(cities))
	for _, city := range cities {
		names = append(names, cityDisplayName(city))
	}
	return strings.Join(names, ", ")
}

func send(bot *tgbotapi.BotAPI, chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if markup != nil {
		msg.ReplyMarkup = markup
	}
	bot.Send(msg)
}

func edit(bot *tgbotapi.BotAPI, chatID int64, msgID int, text string) {
	e := tgbotapi.NewEditMessageText(chatID, msgID, text)
	e.ParseMode = "Markdown"
	bot.Send(e)
}

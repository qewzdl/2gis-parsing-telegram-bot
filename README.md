# 2GIS Parser 🏢

Company parser for 2GIS in Kazakhstan with a Telegram bot.  
**Stack:** Go 1.22 · SQLite · Excel · Telegram Bot API

---

## 📦 Features

- 🔍 Search companies by any query, for example `receipt tape`
- 🏙 Parse by city: Astana, Almaty, Kostanay, Kokshetau, Karaganda, Petropavlovsk
- 📊 Export formatted Excel files
- 💾 Store parsing history in SQLite
- 🤖 Control the workflow through a Telegram bot with inline buttons
- 🐳 Docker support

---

## 🚀 Quick Start

### 1. Get a Telegram Bot Token

Message [@BotFather](https://t.me/BotFather) in Telegram:

```text
/newbot
```

Copy the generated token.

### 2. Configure the App

```bash
cp .env.example .env
nano .env   # add TELEGRAM_TOKEN and DGIS_API_KEY
```

### 3. Run with Docker (recommended)

```bash
docker-compose up --build -d
docker-compose logs -f bot
```

### 4. Run Locally

Requirements: Go 1.22+ and gcc for sqlite3.

```bash
go mod tidy
make run
```

---

## 🤖 Bot Usage

1. Find the bot in Telegram by its username
2. Press `/start`
3. Select **"🔍 Start Parsing"**
4. Enter a query: `receipt tape`
5. Select one, several, or all cities
6. Press **"🚀 Start Parsing"**
7. Receive an Excel file with results

---

## 📁 Project Structure

```text
2gis-parser/
├── cmd/bot/
│   └── main.go           # Telegram bot, dialogs, FSM
├── internal/
│   ├── config/           # Configuration from .env
│   ├── models/           # Data structures
│   ├── parser/           # 2GIS API parser
│   └── storage/          # SQLite + Excel export
├── data/                 # Database files, created automatically
├── exports/              # Excel files, created automatically
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## 📊 Excel Export Format

| No. | Name | City | Address | Phone | Website | Category | Coordinates |
|---|---|---|---|---|---|---|---|
| 1 | Print Solutions LLP | Astana | Republic Ave 10 | +7 7172 ... | ... | Office Supplies | 51.18, 71.44 |

---

## ⚙️ Environment Variables

| Variable | Description | Required |
|---|---|---|
| `TELEGRAM_TOKEN` | Bot token from BotFather | ✅ |
| `DGIS_API_KEY` | API key for 2GIS Places API | ✅ |
| `DGIS_MAX_PAGES` | Maximum result pages to request from 2GIS. Demo keys support up to 5 pages. | ❌ (default: `5`) |
| `DATABASE_PATH` | Path to the SQLite file | ❌ (default: `./data/parser.db`) |
| `OUTPUT_DIR` | Directory for Excel exports | ❌ (default: `./exports`) |

---

## 🔧 Extending

To add a new city:

1. Add it to `SupportedCities` in `internal/models/models.go`
2. Add its region ID to `regionIDBySlug` in `internal/parser/parser.go`
3. Add a button to `buildCityKeyboard` in `cmd/bot/main.go`

---

## 📝 License

MIT

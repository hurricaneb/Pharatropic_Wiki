# Pharatropic Wiki (PTC Wiki)

En snabb, högpresterande och utökbar Wiki-applikation skriven i **Go** med ett komplett **REST API** och ett modernt **React UI**.

## 🚀 Funktioner

- ⚡ **Go (Gin + GORM):** Högpresterande backend API med låg minnesanvändning.
- 🗄️ **SQLite / PostgreSQL:** Typsäker databashantering med automatisk schemamigrering via GORM.
- 📝 **Markdown-stöd:** Inbyggd Markdown-redigerare med live preview.
- 📜 **Full RevisionHistorik:** Varje redigering sparar en revision. Möjlighet att granska tidigare versioner och jämföra diffar.
- 🔍 **Fulltextsökning:** Snabbsökning i alla wikisidor.
- 🔑 **API-nycklar & REST API (`/api/v1`):** Programmatisk åtkomst till alla wikifunktioner.
- 🎨 **Modern Design:** Mörkt/ljust läge, responsiv layout och stilren typografi.

---

## 📡 REST API-dokumentation (`/api/v1`)

| Metod | Slutpunkt | Beskrivning |
| :--- | :--- | :--- |
| `GET` | `/api/v1/health` | Kontrollera API-status |
| `GET` | `/api/v1/pages` | Lista alla wikisidor (`?search=...` & `?tag=...`) |
| `GET` | `/api/v1/pages/:slug` | Hämta en specifik sida via slug |
| `POST` | `/api/v1/pages` | Skapa ny wikisida |
| `PUT` | `/api/v1/pages/:slug` | Uppdatera en sida (skapar ny revision) |
| `DELETE` | `/api/v1/pages/:slug` | Radera en sida |
| `GET` | `/api/v1/pages/:slug/revisions` | Hämta ändringshistorik för en sida |
| `GET` | `/api/v1/search?q=...` | Sök i alla sidor och titlar |

### Exempel på API-anrop

```bash
# Skapa ny sida
curl -X POST http://localhost:8080/api/v1/pages \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Välkommen till Wikin",
    "content": "# Välkommen\nDetta är första sidan!",
    "tags": ["start", "info"]
  }'

# Hämta sida
curl http://localhost:8080/api/v1/pages/valkommen-till-wikin
```

---

## 🛠️ Installation & Körning

### Förutsättningar
- Go 1.22+
- Node.js v20+

### Starta Backend (Go)
```bash
# Ladda ner beroenden och starta servern
go run main.go
```
Backend snurrar på `http://localhost:8080`.

### Starta Frontend (React)
```bash
cd web
npm install
npm run dev
```
Frontend öppnas på `http://localhost:5173`.

---

## 📄 Licens
Projektet är licensierat under [MIT License](LICENSE).

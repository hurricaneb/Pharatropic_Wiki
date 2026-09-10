# Pharatropic Wiki (PTC Wiki)

En blixtsnabb, modern och utökbar Wiki-applikation skriven i **Go** med ett komplett **REST API**, inbyggd **MCP Server (Model Context Protocol)**, automatisk databasmigrering (**SQLite / PostgreSQL**) och ett helintegrerat **React UI**.

---

## 🚀 Funktioner

- ⚡ **Allt-i-ett-server (Go + React):** Go-servern serverar både REST API:t (`/api/v1`), MCP-servern (`/api/v1/mcp`) och det förbyggda React-gränssnittet direkt på samma port (`http://localhost:8080`).
- 🐳 **Docker & TrueNAS SCALE Redo:** Klar för self-hosting med Docker Compose och PostgreSQL 16. Inga portkrockar!
- 🤖 **MCP Server (Model Context Protocol):** Inbyggt stöd för AI-assistenter (Antigravity CLI, Claude Desktop, Cursor, VS Code) att läsa, söka, skapa och uppdatera wikisidor helt nativt.
- 🔒 **Privata & Publika Sidor (`IsPublic`):** Nya sidor skapas som privata (kräver inloggning) som standard. Välj publika sidor för öppen läsning utan konto.
- 👤 **Användarhantering (Admin) & Inloggning:** JWT-baserad inloggning. Endast administratörer kan skapa användarkonton. Varje sidändring registreras med inloggad författare.
- 🔑 **Mina API-nycklar:** Skapa personliga API-nycklar (`ptc_key_...`) med anpassade utgångsdatum (7 dagar, 30 dagar, 90 dagar, 1 år eller Aldrig) och omedelbar återkallning.
- 🔗 **Interna Wiki-länkar:** Skriv `[[Sidtitel]]` eller `[[Sidtitel|Visningstext]]` för att automatiskt skapa interna länkar. Klick på en saknad sida öppnar skaparläget direkt.
- 🔄 **Backlinks ("Sidor som länkar hit"):** Automatisk spårning och visning av alla inkommande länkar till varje wiki-sida.
- 📜 **Versionshistorik & 1-Klick Återställning (Rollback):** Varje redigering sparar en komplett revision. Återställ till tidigare versioner med ett klick.
- 📁 **Filbilagor & Bildhantering:** Ladda upp valfri filtyp direkt till sidor med automatiska Markdown-snippets (`![bild](/uploads/...)`). Bilder och PDF:er visas direkt i webbläsaren; alla andra filtyper laddas ner istället för att köras, som skydd mot skadligt uppladdat innehåll.
- 🔍 **Fulltextsökning & Taggar:** Snabbsökning i titlar och innehåll med tagg-filtrering.
- 🗂️ **Undersidor (en nivå):** Sidor kan ha undersidor, precis som i klassiska wikis — synligt som ett träd i sidofältet, med brödsmulenavigering och en undersides-lista på föräldrasidan. Endast en nivå djupt stöds.

---

## ⚙️ Miljövariabler

| Variabel | Krävs | Beskrivning |
| :--- | :--- | :--- |
| `JWT_SECRET` | Ja (produktion) | Signeringsnyckel för inloggningstokens. Generera med `openssl rand -hex 32`. Saknas den genereras en tillfällig slumpad nyckel vid start, vilket loggar ut alla användare vid varje omstart. |
| `PORT` | Nej | Port servern lyssnar på. Standard `8080`. |
| `DB_DRIVER` | Nej | `sqlite` (standard) eller `postgres`. |
| `DB_PATH` | Nej | Sökväg till SQLite-databasfilen. Standard `wiki.db`. |
| `POSTGRES_HOST` / `POSTGRES_PORT` / `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Om `DB_DRIVER=postgres` | PostgreSQL-anslutningsuppgifter. |
| `API_KEY` | Nej | Valfri master-API-nyckel som ger full åtkomst utan användarkonto. Lämna tom för att bara använda personliga API-nycklar. |

Se `.env.example` för en komplett mall.

> ⚠️ Byt lösenordet för standard-adminkontot (`admin` / `admin`) direkt efter första start.

---

## 🐳 Self-Hosting med Docker Compose & PostgreSQL (TrueNAS SCALE / Portainer)

Wikin levereras med en färdig `docker-compose.yml` anpassad för **TrueNAS SCALE**, Portainer och hemmaservrar.

### 🚀 Starta hela stacken med ett kommandoradsanrop:

```bash
# Sätt en JWT-signeringsnyckel (krävs, se Miljövariabler ovan)
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env

# Starta Wiki + PostgreSQL 16 med persisterad databas & bilagelagring
docker compose up -d
```

Besök därefter **`http://<din-server-ip>:30085`** i webbläsaren!

- **Webbport på servern:** `30085` (Anpassad för TrueNAS SCALE standardintervall `30000-39999`, helt konfigurerbar via `PORT_HOST` i `.env`).
- **PostgreSQL Databas:** Kör helt internt i Docker-nätverket (`db:5432`). Ingen port exponeras mot servern, vilket förhindrar krockar med befintliga databaser eller appar.
- **Volymer:** Sparar automatiskt PostgreSQL-data i `postgres_data` och uppladdade filer i `wiki_uploads`.

---

## 🤖 MCP (Model Context Protocol) Integration

MCP-servern är aktiverad automatiskt på `http://localhost:8080/api/v1/mcp` vid vanliga `go run main.go`.

### 🛠️ Exponerade MCP-verktyg för AI
- `list_pages`: Lista alla wikisidor med status, visningar och taggar.
- `read_page`: Hämta fullständigt innehåll i Markdown för en specifik sida (`slug`).
- `search_pages`: Sök i titlar och innehåll i hela wikin.
- `create_page`: Skapa en ny wikisida med titel, innehåll, sammanfattning, taggar och synlighet.
- `update_page`: Uppdatera en wiki-sida och skapa en ny revision med ändringskommentar.
- `get_backlinks`: Visa alla sidor som länkar till en specifik sida.
- `list_tags`: Hämta alla befintliga taggar.

### ⚙️ Exempel på konfiguration för AI-klienter (Antigravity CLI / Cursor / Claude)

```json
{
  "mcpServers": {
    "pharatropic-wiki": {
      "url": "http://localhost:8080/api/v1/mcp",
      "headers": {
        "X-API-Key": "ptc_key_din_personliga_api_nyckel"
      }
    }
  }
}
```

---

## 📡 REST API Slutpunkter (`/api/v1`)

### 🌐 Öppna Slutpunkter (Publika)
| Metod | Slutpunkt | Beskrivning |
| :--- | :--- | :--- |
| `GET` | `/api/v1/health` | Kontrollera API-status |
| `ANY` | `/api/v1/mcp` | MCP Server (HTTP / SSE Endpoint) |
| `POST` | `/api/v1/auth/login` | Logga in och erhåll JWT Bearer-token |
| `GET` | `/api/v1/pages` | Lista alla publika wikisidor (kräver inloggning för privata) |
| `GET` | `/api/v1/pages/:slug` | Hämta en specifik sida via slug |
| `GET` | `/api/v1/pages/:slug/revisions` | Hämta ändringshistorik för en sida |
| `GET` | `/api/v1/pages/:slug/backlinks` | Hämta alla sidor som länkar till denna sida |
| `GET` | `/api/v1/search?q=...` | Sök i wikisidor och titlar |
| `GET` | `/api/v1/tags` | Lista alla taggar och kategorier |

### 🔒 Behörighetskrävande Slutpunkter (Inloggad eller API-nyckel)
| Metod | Slutpunkt | Beskrivning |
| :--- | :--- | :--- |
| `GET` | `/api/v1/auth/me` | Hämta profil för inloggad användare |
| `POST` | `/api/v1/pages` | Skapa ny wikisida (standard `is_public: false`) |
| `PUT` | `/api/v1/pages/:slug` | Uppdatera en sida (skapar ny revision) |
| `DELETE` | `/api/v1/pages/:slug` | Radera en wiki-sida |
| `POST` | `/api/v1/pages/:slug/attachments` | Ladda upp filbilaga/bild till sida |
| `DELETE` | `/api/v1/attachments/:id` | Radera filbilaga |
| `POST` | `/api/v1/pages/:slug/revert/:id` | Återställ sida till en tidigare revision |
| `GET` | `/api/v1/user/keys` | Lista dina egna API-nycklar |
| `POST` | `/api/v1/user/keys` | Skapa ny personlig API-nyckel |
| `DELETE` | `/api/v1/user/keys/:id` | Återkalla/radera API-nyckel |

### 🛡️ Admin Slutpunkter (Kräver Admin-roll)
| Metod | Slutpunkt | Beskrivning |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/users` | Lista alla registrerade användarkonton |
| `POST` | `/api/v1/admin/users` | Skapa ett nytt användarkonto |

---

## 🛠️ Lokal Utveckling (Utan Docker)

### Förutsättningar
- **Go** v1.22+
- **Node.js** v20+

### Enkel Start (SQLite)
```bash
# Bygg frontend och starta Go-servern med SQLite
(cd web && npm run build) && JWT_SECRET=$(openssl rand -hex 32) go run main.go
```
Besök **`http://localhost:8080`**. Standard Admin: `admin` / `admin`.

---

## 🧪 Köra Tester

```bash
# Backend: alla automatiska tester (handlers, repository, middleware, MCP)
go test -v ./...

# Backend: med testtäckning
go test ./... -coverpkg=./... -coverprofile=cover.out && go tool cover -func=cover.out

# Frontend: komponent- och enhetstester (Vitest + React Testing Library)
cd web && npm test

# Frontend: med testtäckning
cd web && npm run test:coverage
```

---

## 📄 Licens
Projektet är licensierat under [MIT License](LICENSE).

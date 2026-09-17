# 🎓 College Quiz & Academic Assessment Platform

A resilient, high-concurrency assessment engine engineered specifically for large-scale academic environments. Built with **Go**, **MongoDB**, **HTMX**, and **Docker**, the system effortlessly coordinates synchronized examinations across thousands of concurrent students while remaining resource-efficient, distraction-free, and simple to operate.

---

## 🌟 Key Highlights & Philosophy

- **Zero-Contention Append-Only Writes:** During peak exam periods, answers submitted via the "Next" button are recorded as immutable event logs. This eliminates database row locking, prevents race conditions, and guarantees zero lost submissions.
- **Server-Driven Dynamic UI (HTMX):** High-speed interactive experience without heavy client-side JavaScript frameworks. Everything is delivered as fast, lightweight server-rendered HTML.
- **Bilingual by Design (🇸🇦 Arabic & 🇬🇧 English):** Complete native support for English (LTR) and Arabic (RTL), including contextual terminology, dates, numbers, and layout direction.
- **Clean Monochrome Aesthetic:** Distraction-free, minimal interface where color is reserved strictly for meaningful actions and critical state notifications.
- **Hot-Reloading in Development:** Templates and styles update live via Docker volume mounts (`./web:/app/web`) without requiring continuous image rebuilds.

---

## 🏛️ System Architecture

```
                       ┌──────────────────────┐
                       │  Student / Staff UI  │
                       │   (HTML5 + HTMX)     │
                       └──────────┬───────────┘
                                  │ HTTP / JSON / SSE
                                  ▼
                       ┌──────────────────────┐
                       │   Presenter Layer    │
                       │  (Router & Handlers) │
                       └──────────┬───────────┘
                                  │
                                  ▼
                       ┌──────────────────────┐
                       │    Service Layer     │
                       │  (Domain & Business) │
                       └──────────┬───────────┘
                                  │
                                  ▼
                       ┌──────────────────────┐
                       │   Repository Layer   │
                       │ (MongoDB Operations) │
                       └──────────┬───────────┘
                                  │
                                  ▼
                       ┌──────────────────────┐
                       │    MongoDB Cluster   │
                       └──────────────────────┘
```

1. **Presenter Layer (`internal/presenter/`)**:
   - Manages routing using Go's modern standard library `http.ServeMux`.
   - Renders Go templates, handles partial HTMX swaps, and manages locale bundles (`GetI18n`).
2. **Service Layer (`internal/service/`)**:
   - **Auth Service:** Issues and validates stateless, short-lived JWT tokens stored in secure `HttpOnly` cookies.
   - **Quiz Service:** Manages synchronized quiz timing, status transitions, and cohort visibility rules.
   - **Result Service:** Performs atomic final score calculations and generates leaderboards.
   - **Import & Seeder Service:** Handles high-throughput CSV bulk ingestion for rosters up to 25,000+ students.
3. **Repository Layer (`internal/repository/`)**:
   - Manages indexed MongoDB collections (`students`, `quizzes`, `student_answers`, `quiz_results`, `quiz_sessions`).

---

## 🚀 Getting Started

The platform comes fully containerized and ready to run with Docker and a dedicated Makefile.

### 1. Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/)
- `make` (standard on Linux/macOS)

### 2. Start the Application
Simply run:
```bash
make up
```

Once started, the platform will be available at:
👉 **[http://localhost:8090](http://localhost:8090)**

---

## 🔑 Access & Default Credentials

### Staff / Faculty Portal
- **URL:** [http://localhost:8090/admin](http://localhost:8090/admin)
- **Admin PIN:** `admin123`
- **Capabilities:**
  - Create and configure synchronized assessments.
  - Live analytics, grade breakdowns, and student leaderboards.
  - Roster management with sortable registration timestamps.
  - One-click student account activation & deactivation.
  - Bulk CSV student roster imports.

### Student Portal
- **URL:** [http://localhost:8090](http://localhost:8090)
- **Sample Logins (from seeded 25,000 students):**
  - `STU-1A-00001` (Mostafa Shaker — Level 1, Group A)
  - `STU-2B-00002` (Fatima Nabil — Level 2, Group B)
  - `STU-3C-00003` (Dina Kandil — Level 3, Group C)
  - `STU-4D-00004` (Ali Rashad — Level 4, Group D)
  - `STU-5A-00005` (Ibrahim Khalil — Level 5, Group A)

---

## 🛠️ Operational Commands (`Makefile`)

Common tasks are mapped to simple `make` commands:

| Command | Purpose |
| :--- | :--- |
| `make up` | Start app and database in the background |
| `make restart` | Rebuild and restart app container (reflects Go code updates) |
| `make down` | Stop containers gracefully |
| `make logs` | Tail combined logs for all services |
| `make logs-app` | Tail application server logs only |
| `make logs-mongo` | Tail MongoDB database logs |
| `make shell` | Open an interactive bash shell inside the Go app container |
| `make mongo-shell` | Launch MongoDB shell (`mongosh`) to inspect collections |
| `make test` | Run all Go unit and integration tests |
| `make clean` | Stop containers and remove dangling images |
| `make nuke` | Hard reset: removes all containers, networks, and persistent database volumes |

---

## 📂 Project Structure

```
├── cmd/
│   └── server/             # Application entrypoint & dependency injection
├── internal/
│   ├── config/             # Environment variables and system configuration
│   ├── i18n/               # Localization dictionaries (English & Arabic)
│   ├── model/              # Domain structs & database schemas
│   ├── presenter/          # HTTP routers, middlewares, and template handlers
│   ├── repository/         # MongoDB collection queries & bulk operations
│   └── service/            # Core business workflows & grading logic
├── web/
│   ├── static/             # CSS styling, HTMX library, and assets
│   └── templates/          # Go HTML templates (Admin & Student interfaces)
├── test_students_25k.csv   # Production seed dataset (25,000 real student records)
├── docker-compose.yml      # Service definitions with volume mounts
├── Dockerfile              # Multi-stage lightweight Alpine build
└── Makefile                # Developer and administrator task automation
```

---

## 🛡️ License

Built for college and university academic departments. Free to adapt, configure, and deploy.

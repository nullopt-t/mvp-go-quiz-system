# College Quiz System

A high-concurrency, resilient, server-driven web application built for college-scale assessments (5 levels × 4 groups × 500 students = 10,000 students) using **Go**, **HTMX**, **MongoDB**, and **Docker**.

---

## 🏛️ System Architecture & Key Design Decisions

1. **MVP Architecture (Model-View-Presenter)**:
   * **Model (`internal/model`, `internal/repository`)**: MongoDB data layer with index optimizations.
   * **Presenter (`internal/presenter`)**: HTTP handlers, router, server-rendered Go templates & HTMX partial handlers.
   * **Service (`internal/service`)**: Domain logic for JWT authentication, append-only quiz flows, session tracking, and result calculations.

2. **Stateless Short-Lived JWT Authentication**:
   * No cache (e.g. Redis) and no server session store needed.
   * Student login via Student ID (`L1G1-001` .. `L5G4-500`) generates a secure, short-lived JWT containing student ID, cohort level, and group.
   * Stored in secure `HttpOnly` cookies.

3. **Append-Only Write Model (`student_answers`)**:
   * Zero row-lock contention on answer submissions during peak concurrency (thousands of simultaneous students).
   * **"Next" Button Flow**: When a student selects an option and clicks **"Next"**, HTMX issues a `POST /quizzes/{id}/answers` request with `{question_id, answer_id}`.
   * The backend appends a new immutable document in MongoDB without in-place updates.
   * The response returns and renders the next question partial dynamically via HTMX.
   * State reconstruction on page reload or reconnection is derived directly from the append log.
   * Final scores are calculated at submission time by evaluating the latest answer per question against the quiz answer key.

---

## 🚀 Quick Start with Docker

### 1. Run the Entire System
```bash
docker compose up --build
```
The application will start on: `http://localhost:8090`

### 2. Default Accounts for Testing

* **College Hierarchy (10,000 Students seeded across 20 groups)**:
  * Format: `L<Level>G<Group>-<Number>`
  * Examples:
    * Level 1, Group 1: `L1G1-001`, `L1G1-002`, ..., `L1G1-500`
    * Level 2, Group 3: `L2G3-001` .. `L2G3-500`
    * Level 5, Group 4: `L5G4-001` .. `L5G4-500`
* **Staff / Admin Portal**:
  * Admin PIN: `admin123`
  * Portal URL: `http://localhost:8090/admin` (or choose "Staff / Admin" tab on login screen)

---

## 🛠️ Tech Stack

* **Backend**: Go (Go 1.22+ Standard Library Router, `go.mongodb.org/mongo-driver`, `golang-jwt/jwt/v5`)
* **Frontend**: HTML5 + HTMX + Responsive CSS (zero JavaScript framework build step)
* **Database**: MongoDB (running containerized with volume persistence)
* **Containerization**: Multi-stage Docker build & Docker Compose

---

## 🧪 Running Tests

```bash
go test ./... -v
```

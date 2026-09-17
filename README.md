# 🎓 College Quiz System

A lightweight, high-capacity assessment platform built to conduct synchronized examinations across college cohorts without server stress, complex build pipelines, or UI clutter.

---

## 🧠 Mental Model

### 1. How It Works
The system is designed around one core priority: **rock-solid reliability during live exams.**

```
   Students Take Exam                     Faculty Dashboard
 ┌──────────────────────┐               ┌───────────────────────┐
 │ • Quick login by ID  │               │ • Create assessments  │
 │ • Questions one-by-one│               │ • Monitor live status │
 │ • Answers auto-save  │               │ • Instant leaderboards│
 │ • No lost progress   │               │ • Manage student list │
 └──────────┬───────────┘               └───────────┬───────────┘
            │                                       │
            └───────────────────┬───────────────────┘
                                ▼
               ┌─────────────────────────────────┐
               │    Append-Only Exam Engine      │
               │  Zero locking • Instant grading │
               └─────────────────────────────────┘
```

- **Answers are logged, not overwritten:** When thousands of students submit answers simultaneously, the system never fights over database row locks. Every response is logged cleanly, guaranteeing zero dropped submissions even if connection flickers.
- **Server-driven and fast:** No massive client-side frameworks or sluggish page reloads. Every action feels instant and stays lightweight on student devices.
- **Bilingual by default:** Built from the ground up to support both Arabic (🇸🇦 RTL) and English (🇬🇧 LTR) seamlessly.
- **Distraction-free:** Clean, focused monochrome design so students focus entirely on their questions.

---

## 🚀 Quick Start

### Starting the Platform
Everything runs with a single command via Docker:

```bash
make up
```

Once running, open your browser:
👉 **[http://localhost:8090](http://localhost:8090)**

---

## 👥 Usage & Access

### For Faculty & Administrators
- **Login URL:** [http://localhost:8090/admin](http://localhost:8090/admin)
- **Admin PIN:** `admin123`
- **What you can do:**
  - **Create Assessments:** Set duration, target academic levels, target groups, and multiple-choice questions with point weights.
  - **Live Exam Tracking:** View real-time submission metrics, average scores, and student leaderboards.
  - **Student Directory:** Browse, filter, and sort students by code, name, or registration time.
  - **Account Access Control:** Instantly activate or deactivate any student's ability to take exams.
  - **Bulk Import:** Upload student rosters directly via CSV.

### For Students
- **Login URL:** [http://localhost:8090](http://localhost:8090)
- **How to login:** Students sign in using their Student ID (e.g. `STU-1A-00001`).
- **Taking an exam:**
  - Active quizzes scheduled for the student's cohort appear automatically on their dashboard.
  - Answers are saved as they progress through questions.
  - Results and scores are calculated upon submission.

---

## 🛠️ Handy Commands

Manage everything easily using the included `Makefile`:

```bash
make up           # Start the system
make down         # Stop the system
make restart      # Rebuild and apply changes
make logs-app     # View live application logs
make mongo-shell  # Open database console
make nuke         # Reset everything completely
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

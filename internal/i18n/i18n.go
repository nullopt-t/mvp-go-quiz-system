package i18n

import (
	"net/http"
	"strings"
)

type Language string

const (
	LangEN Language = "en"
	LangAR Language = "ar"
)

type TranslationBundle struct {
	Lang Language
	Dir  string // "ltr" or "rtl"
	T    map[string]string
}

var translations = map[Language]map[string]string{
	LangEN: {
		// General & Navbar
		"app_title":       "College Quiz System",
		"nav_stateless":   "Stateless JWT Auth",
		"admin_portal":    "College Administration Portal",
		"logout":          "Log Out",
		"dashboard":       "Dashboard",
		"back_dashboard":  "← Back to Dashboard",
		"admin_mode":      "Admin Mode",
		"lang_switch":     "العربية",
		"lang_switch_url": "?lang=ar",

		// Login
		"welcome_back":        "Welcome Back",
		"login_subtitle":      "Enter your Student ID to access your level and group quizzes",
		"student_id_label":    "Student ID Code",
		"student_id_hint":     "Format: Level 1-5, Group 1-4 (e.g. L1G1-001 to L5G4-500)",
		"login_btn":           "Log In & Access Quizzes",
		"admin_login_title":   "Staff & Faculty Portal",
		"admin_login_sub":     "Enter administrator credentials to manage college quizzes and cohorts",
		"admin_pin_label":     "Staff Access PIN",
		"admin_pin_btn":       "Access Admin Portal",
		"footer_info":         "⚡ Append-Only Quiz Engine • 5 Levels • 20 Groups • 10,000 Students",

		// Student Dashboard
		"assigned_quizzes":    "Assigned Quizzes",
		"dashboard_subtitle":  "Quizzes designated for your academic cohort",
		"level":               "Level",
		"group":               "Group",
		"duration":            "Duration",
		"minutes":             "minutes",
		"questions":           "Questions",
		"start_time":          "Start Time",
		"status_live":         "🔴 LIVE NOW",
		"status_upcoming":     "Upcoming",
		"status_completed":    "Completed",
		"status_expired":      "Expired",
		"btn_enter_live":      "Enter Live Quiz ➔",
		"btn_starts_at":       "Starts at",
		"btn_ended":           "Quiz Ended",
		"btn_view_report":     "View Result Report",
		"no_quizzes":          "No Quizzes Currently Available",
		"no_quizzes_desc":     "Check back later or contact your college department instructor.",
		"past_results":        "Your Past Assessment Results",
		"table_quiz_id":       "Quiz ID",
		"table_score":         "Score",
		"table_correct":       "Correct Answers",
		"table_percentage":    "Percentage",
		"table_submitted_at":  "Submitted At",
		"table_action":        "Action",
		"view_btn":            "View",

		// Live Quiz Room
		"question_of":         "Question",
		"of":                  "of",
		"points":              "Points",
		"pts":                 "pts",
		"saved_notice":        "💾 Answers are saved automatically to the append-only log on progression",
		"btn_next":            "Next Question ➔",
		"btn_submit_final":    "Submit Quiz & View Results ✔",
		"time_expired_alert":  "Time limit reached! Submitting your quiz now...",

		// Quiz Result
		"result_title":        "Quiz Submitted Successfully!",
		"total_score":         "Total Score",
		"accuracy":            "Accuracy",
		"student_label":       "Student",
		"cohort_label":        "Cohort",
		"return_dashboard":    "Return to Dashboard",

		// Admin
		"admin_total_students": "Total Enrolled Students",
		"admin_active_quizzes": "Active Quizzes",
		"admin_create_quiz":    "+ Create New Quiz",
		"admin_hierarchy_title":"College Structure (5 Levels × 4 Groups • 500 Capacity Each)",
		"admin_hierarchy_sub":  "Real-time cohort distribution across all 20 academic sections",
		"admin_enrolled":       "Enrolled",
		"admin_managed_quizzes":"Managed Quizzes",
		"analytics_btn":        "View Analytics 📊",
		"table_target_level":   "Target Level",
		"table_status":         "Status",
		"all_levels":           "All Levels",
	},
	LangAR: {
		// General & Navbar
		"app_title":       "نظام اختبارات الكلية",
		"nav_stateless":   "مصادقة JWT بدون خادم تخزين مؤقت",
		"admin_portal":    "بوابة إدارة الكلية والأساتذة",
		"logout":          "تسجيل الخروج",
		"dashboard":       "لوحة التحكم",
		"back_dashboard":  "← العودة إلى لوحة التحكم",
		"admin_mode":      "وضع الإدارة",
		"lang_switch":     "English",
		"lang_switch_url": "?lang=en",

		// Login
		"welcome_back":        "مرحبًا بك",
		"login_subtitle":      "أدخل الرقم الجامعي للوصول إلى اختبارات مستواك ومجموعتك الدراسية",
		"student_id_label":    "الرقم الجامعي للطالب",
		"student_id_hint":     "الصيغة: المستوى 1-5، المجموعة 1-4 (مثال: L1G1-001 إلى L5G4-500)",
		"login_btn":           "تسجيل الدخول والبدء",
		"admin_login_title":   "بوابة أعضاء هيئة التدريس والإدارة",
		"admin_login_sub":     "أدخل رمز المرور الإداري لإدارة الاختبارات وتتبع نتائج الطلاب",
		"admin_pin_label":     "رمز مرور الإدارة (PIN)",
		"admin_pin_btn":       "دخول لوحة الإدارة",
		"footer_info":         "⚡ محرك اختبارات متزامن • 5 مستويات • 20 مجموعة • 10,000 طالب",

		// Student Dashboard
		"assigned_quizzes":    "الاختبارات المقررة",
		"dashboard_subtitle":  "الاختبارات المخصصة لمستواك الدراسي ومجموعتك الأكاديمية",
		"level":               "المستوى",
		"group":               "المجموعة",
		"duration":            "المدة",
		"minutes":             "دقيقة",
		"questions":           "الأسئلة",
		"start_time":          "وقت البدء",
		"status_live":         "🔴 جاري الآن",
		"status_upcoming":     "قادم",
		"status_completed":    "مكتمل",
		"status_expired":      "منتهي",
		"btn_enter_live":      "دخول الاختبار المباشر ➔",
		"btn_starts_at":       "يبدأ في",
		"btn_ended":           "انتهى الاختبار",
		"btn_view_report":     "عرض تقرير النتيجة",
		"no_quizzes":          "لا توجد اختبارات متاحة حالياً",
		"no_quizzes_desc":     "يرجى مراجعة القسم الأكاديمي أو أستاذ المادة لاحقاً.",
		"past_results":        "نتائج اختباراتك السابقة",
		"table_quiz_id":       "معرّف الاختبار",
		"table_score":         "الدرجة",
		"table_correct":       "الإجابات الصحيحة",
		"table_percentage":    "النسبة المئوية",
		"table_submitted_at":  "تاريخ التسليم",
		"table_action":        "الإجراء",
		"view_btn":            "عرض",

		// Live Quiz Room
		"question_of":         "السؤال",
		"of":                  "من",
		"points":              "الدرجات",
		"pts":                 "درجة",
		"saved_notice":        "💾 يتم حفظ الإجابات تلقائياً وبشكل فوري عند الانتقال للسؤال التالي",
		"btn_next":            "السؤال التالي ➔",
		"btn_submit_final":    "تسليم الاختبار وعرض النتيجة ✔",
		"time_expired_alert":  "انتهى الوقت المحدد! جاري تسليم إجاباتك تلقائياً...",

		// Quiz Result
		"result_title":        "تم تسليم الاختبار بنجاح!",
		"total_score":         "مجموع الدرجات",
		"accuracy":            "نسبة النجاح",
		"student_label":       "الطالب",
		"cohort_label":        "المستوى والمجموعة",
		"return_dashboard":    "العودة للوحة التحكم",

		// Admin
		"admin_total_students": "إجمالي الطلاب المسجلين",
		"admin_active_quizzes": "الاختبارات النشطة",
		"admin_create_quiz":    "+ إنشاء اختبار جديد",
		"admin_hierarchy_title":"هيكل الكلية (5 مستويات × 4 مجموعات • سعة 500 طالب لكل مجموعة)",
		"admin_hierarchy_sub":  "توزيع الطلاب الفعلي عبر كافة الأقسام العشرين",
		"admin_enrolled":       "المسجلون",
		"admin_managed_quizzes":"الاختبارات المسجلة",
		"analytics_btn":        "عرض الإحصائيات 📊",
		"table_target_level":   "المستوى المستهدف",
		"table_status":         "الحالة",
		"all_levels":           "كافة المستويات",
	},
}

func GetLanguage(r *http.Request) Language {
	// 1. Query parameter preference
	if qLang := r.URL.Query().Get("lang"); qLang != "" {
		if strings.ToLower(qLang) == "ar" {
			return LangAR
		}
		return LangEN
	}

	// 2. Cookie preference
	if cookie, err := r.Cookie("lang_pref"); err == nil && cookie != nil {
		if cookie.Value == "ar" {
			return LangAR
		}
		if cookie.Value == "en" {
			return LangEN
		}
	}

	// 3. Accept-Language header fallback
	acceptLang := r.Header.Get("Accept-Language")
	if strings.HasPrefix(strings.ToLower(acceptLang), "ar") {
		return LangAR
	}

	return LangEN
}

func GetBundle(lang Language) TranslationBundle {
	dir := "ltr"
	if lang == LangAR {
		dir = "rtl"
	}

	dict, ok := translations[lang]
	if !ok {
		dict = translations[LangEN]
		dir = "ltr"
	}

	return TranslationBundle{
		Lang: lang,
		Dir:  dir,
		T:    dict,
	}
}

func (b TranslationBundle) Translate(key string) string {
	if val, ok := b.T[key]; ok {
		return val
	}
	return key
}

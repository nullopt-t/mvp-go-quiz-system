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
		"app_title":       "College Assessment Portal",
		"nav_stateless":   "Official Portal",
		"admin_portal":    "Staff & Administration Portal",
		"logout":          "Log Out",
		"dashboard":       "Dashboard",
		"back_dashboard":  "Back to Dashboard",
		"admin_mode":      "Admin Mode",
		"lang_switch":     "العربية",
		"lang_switch_url": "?lang=ar",

		// Login
		"welcome_back":        "Welcome Back",
		"login_subtitle":      "Enter your Student ID to access your scheduled examinations",
		"student_id_label":    "Student ID",
		"student_id_hint":     "Format: Level 1-5, Group 1-4 (e.g. L1G1-001)",
		"login_btn":           "Sign In",
		"admin_login_title":   "Staff & Faculty Portal",
		"admin_login_sub":     "Enter administrator credentials to access faculty controls",
		"admin_pin_label":     "Access PIN",
		"admin_pin_btn":       "Sign In to Portal",
		"footer_info":         "College Academic Assessment System",

		// Student Dashboard
		"assigned_quizzes":    "Assigned Assessments",
		"dashboard_subtitle":  "Examinations assigned to your cohort",
		"level":               "Level",
		"group":               "Group",
		"duration":            "Duration",
		"minutes":             "minutes",
		"questions":           "Questions",
		"start_time":          "Scheduled Start",
		"status_live":         "LIVE",
		"status_upcoming":     "Upcoming",
		"status_completed":    "Completed",
		"status_expired":      "Ended",
		"btn_enter_live":      "Start Assessment",
		"btn_starts_at":       "Starts at",
		"btn_ended":           "Assessment Ended",
		"btn_view_report":     "View Score Report",
		"no_quizzes":          "No Assessments Currently Available",
		"no_quizzes_desc":     "There are no scheduled assessments for your section at this time.",
		"past_results":        "Your Assessment History",
		"table_quiz_name":     "Assessment Title",
		"table_score":         "Score",
		"table_correct":       "Correct Answers",
		"table_percentage":    "Grade",
		"table_submitted_at":  "Submission Date",
		"table_action":        "Details",
		"view_btn":            "View Report",

		// Live Quiz Room
		"question_of":         "Question",
		"of":                  "of",
		"points":              "Points",
		"pts":                 "pts",
		"saved_notice":        "Your answers are saved automatically as you proceed",
		"btn_next":            "Next Question",
		"btn_submit_final":    "Submit Assessment",
		"time_expired_alert":  "Time limit reached. Your assessment is being submitted automatically.",

		// Quiz Result
		"result_title":        "Assessment Submitted Successfully",
		"total_score":         "Final Score",
		"accuracy":            "Accuracy",
		"student_label":       "Student",
		"cohort_label":        "Academic Cohort",
		"return_dashboard":    "Return to Dashboard",

		// Admin
		"admin_total_students": "Total Enrolled Students",
		"admin_active_quizzes": "Active Assessments",
		"admin_create_quiz":    "Create New Assessment",
		"admin_hierarchy_title":"College Cohort Overview (5 Levels, 4 Groups)",
		"admin_hierarchy_sub":  "Student enrollment distribution across all academic sections",
		"admin_enrolled":       "Enrolled Students",
		"admin_managed_quizzes":"Active and Scheduled Assessments",
		"analytics_btn":        "View Results",
		"table_target_level":   "Target Level",
		"table_status":         "Status",
		"all_levels":           "All Levels",
	},
	LangAR: {
		// General & Navbar
		"app_title":       "بوابة الاختبارات الأكاديمية",
		"nav_stateless":   "البوابة الرسمية",
		"admin_portal":    "بوابة أعضاء هيئة التدريس والإدارة",
		"logout":          "تسجيل الخروج",
		"dashboard":       "لوحة التحكم",
		"back_dashboard":  "العودة إلى لوحة التحكم",
		"admin_mode":      "وضع الإدارة",
		"lang_switch":     "English",
		"lang_switch_url": "?lang=en",

		// Login
		"welcome_back":        "مرحبًا بك",
		"login_subtitle":      "أدخل الرقم الجامعي للوصول إلى الاختبارات المقررة لمستواك ومجموعتك",
		"student_id_label":    "الرقم الجامعي للطالب",
		"student_id_hint":     "الصيغة: المستوى 1-5، المجموعة 1-4 (مثال: L1G1-001)",
		"login_btn":           "تسجيل الدخول",
		"admin_login_title":   "بوابة أعضاء هيئة التدريس والإدارة",
		"admin_login_sub":     "أدخل رمز المرور للوصول إلى أدوات إدارة الاختبارات",
		"admin_pin_label":     "رمز المرور",
		"admin_pin_btn":       "دخول البوابة",
		"footer_info":         "نظام التقييم والاختبارات الجامعية",

		// Student Dashboard
		"assigned_quizzes":    "الاختبارات المقررة",
		"dashboard_subtitle":  "الاختبارات المخصصة لمستواك الدراسي ومجموعتك",
		"level":               "المستوى",
		"group":               "المجموعة",
		"duration":            "المدة",
		"minutes":             "دقيقة",
		"questions":           "عدد الأسئلة",
		"start_time":          "موعد البدء",
		"status_live":         "جاري الآن",
		"status_upcoming":     "قادم",
		"status_completed":    "مكتمل",
		"status_expired":      "منتهي",
		"btn_enter_live":      "بدء الاختبار",
		"btn_starts_at":       "يبدأ في",
		"btn_ended":           "انتهى الاختبار",
		"btn_view_report":     "عرض تقرير النتيجة",
		"no_quizzes":          "لا توجد اختبارات متاحة حالياً",
		"no_quizzes_desc":     "لا توجد اختبارات مجدولة لمجموعتك في الوقت الحالي.",
		"past_results":        "سجل الاختبارات السابقة",
		"table_quiz_name":     "عنوان الاختبار",
		"table_score":         "الدرجة",
		"table_correct":       "الإجابات الصحيحة",
		"table_percentage":    "النسبة المئوية",
		"table_submitted_at":  "تاريخ التسليم",
		"table_action":        "التفاصيل",
		"view_btn":            "عرض التقرير",

		// Live Quiz Room
		"question_of":         "السؤال",
		"of":                  "من",
		"points":              "الدرجات",
		"pts":                 "درجة",
		"saved_notice":        "يتم حفظ إجاباتك تلقائياً عند الانتقال بين الأسئلة",
		"btn_next":            "السؤال التالي",
		"btn_submit_final":    "تسليم الاختبار",
		"time_expired_alert":  "انتهى الوقت المحدد للاختبار. جاري تسليم إجاباتك تلقائياً.",

		// Quiz Result
		"result_title":        "تم تسليم الاختبار بنجاح",
		"total_score":         "الدرجة النهائية",
		"accuracy":            "نسبة النجاح",
		"student_label":       "الطالب",
		"cohort_label":        "المستوى والمجموعة",
		"return_dashboard":    "العودة للوحة التحكم",

		// Admin
		"admin_total_students": "إجمالي الطلاب المسجلين",
		"admin_active_quizzes": "الاختبارات النشطة",
		"admin_create_quiz":    "إنشاء اختبار جديد",
		"admin_hierarchy_title":"هيكل الكلية (5 مستويات، 4 مجموعات)",
		"admin_hierarchy_sub":  "توزيع الطلاب الفعلي عبر كافة الأقسام الدراسية",
		"admin_enrolled":       "الطلاب المقيدون",
		"admin_managed_quizzes":"الاختبارات المسجلة والمجدولة",
		"analytics_btn":        "عرض النتائج",
		"table_target_level":   "المستوى المستهدف",
		"table_status":         "الحالة",
		"all_levels":           "كافة المستويات",
	},
}

func GetLanguage(r *http.Request) Language {
	if qLang := r.URL.Query().Get("lang"); qLang != "" {
		if strings.ToLower(qLang) == "ar" {
			return LangAR
		}
		return LangEN
	}

	if cookie, err := r.Cookie("lang_pref"); err == nil && cookie != nil {
		if cookie.Value == "ar" {
			return LangAR
		}
		if cookie.Value == "en" {
			return LangEN
		}
	}

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

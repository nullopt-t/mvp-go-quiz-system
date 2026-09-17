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
		// Navbar & General
		"app_title":       "College Quizzes",
		"admin_portal":    "Admin Portal",
		"logout":          "Log Out",
		"dashboard":       "Dashboard",
		"back_dashboard":  "Back to Dashboard",
		"admin_mode":      "Admin Mode",
		"lang_switch":     "العربية",
		"lang_switch_url": "?lang=ar",

		// Student Login
		"welcome_back":        "Welcome",
		"login_subtitle":      "Enter your Student ID to view your quizzes",
		"student_id_label":    "Student ID",
		"login_btn":           "Log In",
		"err_student_id_req":  "Please enter your Student ID",
		"err_student_not_found": "Student ID not found or inactive",

		// Admin Login
		"admin_login_title":   "Staff Login",
		"admin_login_sub":     "Sign in to manage quizzes and view student results",
		"admin_pin_label":     "Admin PIN",
		"admin_pin_btn":       "Log In",
		"err_admin_pin_invalid": "Invalid PIN code",

		// Student Dashboard
		"assigned_quizzes":    "Your Quizzes",
		"dashboard_subtitle":  "Quizzes available for your level and group",
		"level":               "Level",
		"group":               "Group",
		"duration":            "Duration",
		"minutes":             "minutes",
		"questions":           "Questions",
		"start_time":          "Start Time",
		"status_live":         "LIVE",
		"status_upcoming":     "Upcoming",
		"status_completed":    "Completed",
		"status_expired":      "Ended",
		"btn_enter_live":      "Start Quiz",
		"btn_starts_at":       "Starts at",
		"btn_ended":           "Quiz Ended",
		"btn_view_report":     "View Results",
		"no_quizzes":          "No Quizzes Available",
		"no_quizzes_desc":     "There are no quizzes scheduled for your group right now.",
		"past_results":        "Past Results",
		"table_quiz_name":     "Quiz",
		"table_score":         "Score",
		"table_correct":       "Correct",
		"table_percentage":    "Grade",
		"table_submitted_at":  "Date",
		"table_action":        "Details",
		"view_btn":            "View",

		// Live Quiz Room
		"question_of":         "Question",
		"of":                  "of",
		"points":              "Points",
		"pts":                 "pts",
		"saved_notice":        "Your answers are saved automatically as you go",
		"btn_next":            "Next Question",
		"btn_submit_final":    "Finish & Submit",
		"time_expired_alert":  "Time is up! Submitting your answers now.",

		// Quiz Result
		"result_title":        "Quiz Submitted!",
		"total_score":         "Your Score",
		"accuracy":            "Accuracy",
		"student_label":       "Student",
		"cohort_label":        "Cohort",
		"return_dashboard":    "Back to Dashboard",

		// Admin
		"admin_total_students": "Total Students",
		"admin_active_quizzes": "Active Quizzes",
		"admin_create_quiz":    "New Quiz",
		"admin_hierarchy_title":"Cohort Overview (5 Levels, 4 Groups: A, B, C, D)",
		"admin_hierarchy_sub":  "Student enrollment per group",
		"admin_enrolled":       "Students",
		"admin_managed_quizzes":"Quizzes",
		"admin_import_title":   "Import Students (CSV)",
		"admin_import_sub":     "Upload a CSV roster to bulk register or update students",
		"admin_import_btn":     "Import CSV",
		"admin_sample_csv":     "Download Sample CSV",
		"admin_students_title": "Students Roster",
		"admin_students_sub":   "Browse and manage registered students by cohort",
		"admin_add_student":    "Add Student",
		"admin_student_profile":"Student Profile",
		"admin_filter_cohort":  "Filter Cohort",
		"all_groups":           "All Groups",
		"filter_btn":           "Filter",
		"sort_by_label":        "Sort By",
		"sort_time":            "Registration Time",
		"sort_name":            "Student Name",
		"sort_id":              "Student ID",
		"sort_asc":             "Ascending",
		"sort_desc":            "Descending",
		"student_created_success": "Student saved successfully",
		"analytics_btn":        "View Results",
		"table_target_level":   "Target Level",
		"table_status":         "Status",
		"all_levels":           "All Levels",
		"admin_preview_title":  "Review Imported Students",
		"admin_preview_sub":    "Verify or edit student information before confirming import to the database",
		"admin_confirm_import": "Confirm & Import Students",
		"admin_cancel_import":  "Cancel",
		"admin_remove_row":     "Remove",
		"admin_preview_btn":    "Review & Import",
		"admin_preview_count":  "Students to Import",
		"admin_preview_notice": "You can modify student code, name, level, or group inline before confirming import.",
		"admin_preview_showing": "Showing sample of %d students out of %d total in file.",
		"admin_confirm_all_import": "Import All %d Students from File",
		"admin_search_preview_placeholder": "Search by ID or name...",
		"admin_search_filter_btn": "Search & Filter",
		"admin_clear_filter": "Reset",
		"admin_dashboard_badge": "Faculty & Exam Control System",
		"admin_dashboard_welcome": "Welcome, Administrator",
		"admin_dashboard_sub": "Central management console for exams, cohorts, and academic rosters",
		"admin_cohort_summary": "Active Cohorts",
		"admin_view_roster_btn": "Browse All Students",
		"admin_drop_csv_hint": "Select or drag student roster CSV (UTF-8)",
		"admin_browse_files": "Browse File",
		"admin_drag_hint": "or drag and drop here",
		"admin_no_file_selected": "No file chosen",
		"admin_quick_actions": "Quick Actions",
		"recent_students": "Recent Students",
		"recent_students_sub": "Quick view of registered students with direct access to actions and profile",
		"admin_create_quiz_title": "Create New Academic Assessment",
		"admin_create_quiz_sub": "Configure assessment settings, cohort criteria, and questions with answer keys",
		"quiz_title_label": "Assessment Title",
		"quiz_title_placeholder": "e.g., Computer Science Midterm Examination",
		"quiz_desc_label": "Instructions / Description",
		"quiz_desc_placeholder": "Enter clear instructions for students...",
		"quiz_target_level": "Target Level",
		"quiz_target_groups": "Target Groups (Optional)",
		"quiz_target_groups_placeholder": "e.g., A, B (Leave blank for all groups)",
		"quiz_start_time_label": "Official Exam Start Time",
		"quiz_duration_minutes": "Time Limit (Minutes)",
		"quiz_questions_bank": "Questions Bank",
		"quiz_question_num": "Question %d",
		"quiz_points_label": "Points",
		"quiz_statement_placeholder": "Enter question statement...",
		"quiz_option_num": "Option %d",
		"quiz_option_placeholder": "Option %s text",
		"quiz_correct_key": "Correct Answer Key",
		"quiz_add_question_btn": "Add Another Question",
		"quiz_remove_question_btn": "Delete Question",
		"quiz_save_publish_btn": "Save & Publish Assessment",

		// Status badges
		"status_active":   "Active",
		"status_inactive": "Inactive",

		// Analytics page
		"analytics_sub":            "Performance Breakdown and Leaderboard",
		"analytics_submissions":    "Completed Submissions",
		"analytics_avg_score":      "Average Score",
		"analytics_top_score":      "Highest Score",
		"analytics_leaderboard":    "Student Submissions Leaderboard",
		"analytics_rank":           "Rank",
		"analytics_correct_total":  "Correct / Total",
		"analytics_submitted_at":   "Completion Time",
		"analytics_no_submissions": "No submissions recorded yet for this quiz.",

		// Admin dashboard
		"admin_no_cohorts":  "No student cohorts registered yet. Upload a CSV roster to populate cohorts.",
		"admin_quizzes_sub": "Overview of assessments, active timeframes, and results tracking",
		"admin_no_quizzes":  "No quizzes created yet.",

		// Admin student management
		"admin_add_student_sub":         "Register a new student or update an existing record by student code.",
		"admin_student_name_placeholder": "Full Student Name",
		"admin_registered_at":            "Registered At",
		"admin_completed_quizzes":        "Completed Quizzes",
		"admin_no_student_submissions":   "No quiz submissions recorded for this student yet.",
		"admin_no_students_filter":       "No students found for the selected filters.",

		// Student activation
		"btn_activate":                 "Activate",
		"btn_deactivate":               "Deactivate",
		"student_status_updated":       "Student status updated successfully",
		"is_active_label":              "Account Status",
		"active_option":                "Active (Allowed to take quizzes)",
		"inactive_option":              "Inactive (Blocked from taking quizzes)",

		// Quiz creation
		"quiz_questions_bank_sub": "Configure multiple choice questions, individual point weights, and correct options.",
	},
	LangAR: {
		// Navbar & General
		"app_title":       "اختبارات الكلية",
		"admin_portal":    "لوحة الإدارة",
		"logout":          "تسجيل الخروج",
		"dashboard":       "لوحة التحكم",
		"back_dashboard":  "العودة للوحة التحكم",
		"admin_mode":      "وضع الإدارة",
		"lang_switch":     "English",
		"lang_switch_url": "?lang=en",

		// Student Login
		"welcome_back":        "مرحباً بك",
		"login_subtitle":      "أدخل رقمك الجامعي لبدء اختباراتك",
		"student_id_label":    "الرقم الجامعي",
		"login_btn":           "دخول",
		"err_student_id_req":  "يرجى إدخال الرقم الجامعي",
		"err_student_not_found": "الرقم الجامعي غير مسجل أو غير مفعل",

		// Admin Login
		"admin_login_title":   "دخول الأساتذة والإدارة",
		"admin_login_sub":     "تسجيل الدخول لإدارة الاختبارات ومتابعة النتائج",
		"admin_pin_label":     "رمز المرور",
		"admin_pin_btn":       "دخول",
		"err_admin_pin_invalid": "رمز المرور غير صحيح",

		// Student Dashboard
		"assigned_quizzes":    "اختباراتك",
		"dashboard_subtitle":  "الاختبارات المتاحة لمستواك ومجموعتك",
		"level":               "المستوى",
		"group":               "المجموعة",
		"duration":            "المدة",
		"minutes":             "دقيقة",
		"questions":           "عدد الأسئلة",
		"start_time":          "وقت البدء",
		"status_live":         "متاح الآن",
		"status_upcoming":     "قادم",
		"status_completed":    "مكتمل",
		"status_expired":      "منتهي",
		"btn_enter_live":      "بدء الاختبار",
		"btn_starts_at":       "يبدأ في",
		"btn_ended":           "انتهى الاختبار",
		"btn_view_report":     "عرض النتيجة",
		"no_quizzes":          "لا توجد اختبارات حالياً",
		"no_quizzes_desc":     "لا توجد اختبارات مجدولة لمجموعتك في الوقت الحالي.",
		"past_results":        "نتائجك السابقة",
		"table_quiz_name":     "الاختبار",
		"table_score":         "الدرجة",
		"table_correct":       "الإجابات الصحيحة",
		"table_percentage":    "النسبة",
		"table_submitted_at":  "التاريخ",
		"table_action":        "التفاصيل",
		"view_btn":            "عرض",

		// Live Quiz Room
		"question_of":         "السؤال",
		"of":                  "من",
		"points":              "الدرجات",
		"pts":                 "درجة",
		"saved_notice":        "يتم حفظ إجاباتك تلقائياً أثناء الحل",
		"btn_next":            "السؤال التالي",
		"btn_submit_final":    "تسليم الاختبار",
		"time_expired_alert":  "انتهى الوقت المحدد! جاري تسليم إجاباتك الآن.",

		// Quiz Result
		"result_title":        "تم تسليم الاختبار بنجاح!",
		"total_score":         "درجتك",
		"accuracy":            "نسبة النجاح",
		"student_label":       "الطالب",
		"cohort_label":        "المستوى والمجموعة",
		"return_dashboard":    "العودة للاختبارات",

		// Admin
		"admin_total_students": "إجمالي الطلاب",
		"admin_active_quizzes": "الاختبارات النشطة",
		"admin_create_quiz":    "إنشاء اختبار جديد",
		"admin_hierarchy_title":"توزيع الطلاب (5 مستويات، 4 مجموعات: A, B, C, D)",
		"admin_hierarchy_sub":  "عدد الطلاب المقيدين في كل مجموعة",
		"admin_enrolled":       "الطلاب",
		"admin_managed_quizzes":"الاختبارات",
		"admin_import_title":   "استيراد بيانات الطلاب (CSV)",
		"admin_import_sub":     "رفع ملف CSV لتسجيل أو تحديث بيانات الطلاب دفعة واحدة",
		"admin_import_btn":     "استيراد الملف",
		"admin_sample_csv":     "تحميل نموذج الملف (CSV)",
		"admin_students_title": "قائمة الطلاب",
		"admin_students_sub":   "عرض وإدارة الطلاب المسجلين حسب المستوى والمجموعة",
		"admin_add_student":    "إضافة طالب",
		"admin_student_profile":"الملف الأكاديمي للطالب",
		"admin_filter_cohort":  "تصفية المجموعة",
		"all_groups":           "كافة المجموعات",
		"filter_btn":           "تصفية",
		"sort_by_label":        "ترتيب حسب",
		"sort_time":            "وقت إضافة الطالب",
		"sort_name":            "اسم الطالب",
		"sort_id":              "الرقم الجامعي",
		"sort_asc":             "تصاعدي",
		"sort_desc":            "تنازلي",
		"student_created_success": "تم حفظ بيانات الطالب بنجاح",
		"analytics_btn":        "النتائج والإحصائيات",
		"table_target_level":   "المستوى المستهدف",
		"table_status":         "الحالة",
		"all_levels":           "كافة المستويات",
		"admin_preview_title":  "مراجعة بيانات الطلاب قبل الاستيراد",
		"admin_preview_sub":    "يمكنك مراجعة وتعديل بيانات الطلاب قبل تأكيد حفظها في النظام",
		"admin_confirm_import": "تأكيد واستيراد الطلاب",
		"admin_cancel_import":  "إلغاء",
		"admin_remove_row":     "حذف",
		"admin_preview_btn":    "معاينة واستيراد",
		"admin_preview_count":  "عدد الطلاب الجاهزين للاستيراد",
		"admin_preview_notice": "يمكنك تعديل الرقم الجامعي، الاسم، المستوى، والمجموعة مباشرة في الجدول أدناه قبل الضغط على تأكيد الاستيراد.",
		"admin_preview_showing": "يتم عرض عينة مكونة من %d طالباً من إجمالي %d طالباً في الملف.",
		"admin_confirm_all_import": "استيراد كافة الطلاب من الملف (%d طالب)",
		"admin_search_preview_placeholder": "بحث بالرقم الجامعي أو الاسم...",
		"admin_search_filter_btn": "بحث وتصفية",
		"admin_clear_filter": "إلغاء التصفية",
		"admin_dashboard_badge": "نظام كنترول الاختبارات وشؤون الطلاب",
		"admin_dashboard_welcome": "مرحباً، إدارة الكلية والكنترول",
		"admin_dashboard_sub": "لوحة الإدارة المركزية لمتابعة الاختبارات والمجموعات وقوائم الطلاب",
		"admin_cohort_summary": "المجموعات النشطة",
		"admin_view_roster_btn": "استعراض سجل الطلاب الكامل",
		"admin_drop_csv_hint": "اختر ملف CSV الخاص بقوائم الطلاب (ترميز UTF-8)",
		"admin_browse_files": "اختيار ملف",
		"admin_drag_hint": "أو قم بسحب الملف وإفلاته هنا",
		"admin_no_file_selected": "لم يتم اختيار ملف بعد",
		"admin_quick_actions": "إجراءات سريعة",
		"recent_students": "أحدث الطلاب المسجلين",
		"recent_students_sub": "عرض سريع لبيانات الطلاب مع إمكانية عرض الملف وإدارة حالة الحساب",
		"admin_create_quiz_title": "إنشاء اختبار أكاديمي جديد",
		"admin_create_quiz_sub": "تحديد إعدادات الاختبار، المجموعات والمستويات المستهدفة، وإدخال الأسئلة مع مفاتيح الإجابات",
		"quiz_title_label": "عنوان الاختبار",
		"quiz_title_placeholder": "مثال: الاختبار النصفي لمادة علوم الحاسب",
		"quiz_desc_label": "التعليمات / الوصف",
		"quiz_desc_placeholder": "اكتب تعليمات وتوجيهات واضحة للطلاب...",
		"quiz_target_level": "المستوى المستهدف",
		"quiz_target_groups": "المجموعات المستهدفة (اختياري)",
		"quiz_target_groups_placeholder": "مثال: A, B (اتركه فارغاً لجميع المجموعات)",
		"quiz_start_time_label": "موعد بدء الاختبار الرسمي",
		"quiz_duration_minutes": "مدة الاختبار للطالب (بالدقائق)",
		"quiz_questions_bank": "بنك الأسئلة",
		"quiz_question_num": "السؤال %d",
		"quiz_points_label": "الدرجات",
		"quiz_statement_placeholder": "اكتب نص السؤال هنا...",
		"quiz_option_num": "الخيار %d",
		"quiz_option_placeholder": "نص الخيار %s",
		"quiz_correct_key": "مفتاح الإجابة الصحيحة",
		"quiz_add_question_btn": "إضافة سؤال جديد",
		"quiz_remove_question_btn": "حذف السؤال",
		"quiz_save_publish_btn": "حفظ ونشر الاختبار",

		// Status badges
		"status_active":   "نشط",
		"status_inactive": "غير نشط",

		// Analytics page
		"analytics_sub":            "تحليل الأداء وترتيب الطلاب",
		"analytics_submissions":    "إجمالي التسليمات",
		"analytics_avg_score":      "متوسط الدرجات",
		"analytics_top_score":      "أعلى درجة",
		"analytics_leaderboard":    "ترتيب الطلاب حسب الدرجات",
		"analytics_rank":           "الترتيب",
		"analytics_correct_total":  "الصحيح / الإجمالي",
		"analytics_submitted_at":   "وقت التسليم",
		"analytics_no_submissions": "لا توجد تسليمات مسجلة لهذا الاختبار بعد.",

		// Admin dashboard
		"admin_no_cohorts":  "لا توجد مجموعات مسجلة بعد. قم برفع ملف CSV لتسجيل الطلاب.",
		"admin_quizzes_sub": "نظرة عامة على الاختبارات والجداول الزمنية وتتبع النتائج",
		"admin_no_quizzes":  "لا توجد اختبارات بعد.",

		// Admin student management
		"admin_add_student_sub":         "تسجيل طالب جديد أو تحديث بيانات طالب موجود عن طريق الرقم الجامعي.",
		"admin_student_name_placeholder": "الاسم الكامل للطالب",
		"admin_registered_at":            "تاريخ التسجيل",
		"admin_completed_quizzes":        "الاختبارات المكتملة",
		"admin_no_student_submissions":   "لا توجد تسليمات مسجلة لهذا الطالب بعد.",
		"admin_no_students_filter":       "لا يوجد طلاب مطابقون للتصفية المحددة.",

		// Student activation
		"btn_activate":                 "تفعيل الحساب",
		"btn_deactivate":               "تعطيل الحساب",
		"student_status_updated":       "تم تحديث حالة حساب الطالب بنجاح",
		"is_active_label":              "حالة الحساب",
		"active_option":                "نشط (مسموح له بأداء الاختبارات)",
		"inactive_option":              "غير نشط (محظور من الاختبارات)",

		// Quiz creation
		"quiz_questions_bank_sub": "أضف أسئلة الاختيار من متعدد مع الدرجات ومفاتيح الإجابات الصحيحة.",
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

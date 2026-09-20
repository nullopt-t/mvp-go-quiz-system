package presenter

import (
	"html/template"
	"net/http"
	"strings"

	"quiz-system/internal/service"
)

type AuthPresenter struct {
	authService service.AuthService
	templates   *template.Template
}

func NewAuthPresenter(authService service.AuthService, tmpl *template.Template) *AuthPresenter {
	return &AuthPresenter{
		authService: authService,
		templates:   tmpl,
	}
}

// Student Login
func (p *AuthPresenter) RenderLogin(w http.ResponseWriter, r *http.Request) {
	errParam, _ := GetFlashMessages(w, r)
	i18nBundle := GetI18n(r)
	data := map[string]interface{}{
		"Error": errParam,
		"I18n":  i18nBundle,
	}
	_ = p.templates.ExecuteTemplate(w, "login.html", data)
}

func (p *AuthPresenter) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	i18nBundle := GetI18n(r)
	studentCode := strings.ToUpper(strings.TrimSpace(r.FormValue("student_code")))
	if studentCode == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = p.templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Error": i18nBundle.Translate("err_student_id_req"),
			"I18n":  i18nBundle,
		})
		return
	}

	token, _, err := p.authService.LoginStudent(r.Context(), studentCode)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = p.templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Error": i18nBundle.Translate("err_student_not_found"),
			"I18n":  i18nBundle,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/quizzes", http.StatusSeeOther)
}

// Admin / Staff Login (Separate Route)
func (p *AuthPresenter) RenderAdminLogin(w http.ResponseWriter, r *http.Request) {
	errParam, _ := GetFlashMessages(w, r)
	i18nBundle := GetI18n(r)
	data := map[string]interface{}{
		"Error": errParam,
		"I18n":  i18nBundle,
	}
	_ = p.templates.ExecuteTemplate(w, "admin_login.html", data)
}

func (p *AuthPresenter) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	i18nBundle := GetI18n(r)
	pin := strings.TrimSpace(r.FormValue("pin"))
	token, err := p.authService.LoginAdmin(pin)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = p.templates.ExecuteTemplate(w, "admin_login.html", map[string]interface{}{
			"Error": i18nBundle.Translate("err_admin_pin_invalid"),
			"I18n":  i18nBundle,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (p *AuthPresenter) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (p *AuthPresenter) HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

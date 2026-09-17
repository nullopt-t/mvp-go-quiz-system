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

func (p *AuthPresenter) RenderLogin(w http.ResponseWriter, r *http.Request) {
	errParam := r.URL.Query().Get("error")
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

	loginType := r.FormValue("login_type")
	if loginType == "admin" {
		pin := strings.TrimSpace(r.FormValue("pin"))
		token, err := p.authService.LoginAdmin(pin)
		if err != nil {
			http.Redirect(w, r, "/login?error=Invalid+Admin+PIN", http.StatusSeeOther)
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
		return
	}

	studentCode := strings.ToUpper(strings.TrimSpace(r.FormValue("student_code")))
	if studentCode == "" {
		http.Redirect(w, r, "/login?error=Student+ID+is+required", http.StatusSeeOther)
		return
	}

	token, _, err := p.authService.LoginStudent(r.Context(), studentCode)
	if err != nil {
		http.Redirect(w, r, "/login?error=Student+ID+not+found+or+inactive", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

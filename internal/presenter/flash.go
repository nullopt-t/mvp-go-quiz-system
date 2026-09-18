package presenter

import (
	"net/http"
	"net/url"
)

const (
	FlashErrorCookie   = "flash_error"
	FlashSuccessCookie = "flash_success"
)

// SetFlashError sets a one-time flash error message cookie
func SetFlashError(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     FlashErrorCookie,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   60, // 1 minute
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetFlashSuccess sets a one-time flash success message cookie
func SetFlashSuccess(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     FlashSuccessCookie,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   60, // 1 minute
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetFlashMessages retrieves and clears one-time flash messages from cookies
// Falling back to query parameters if legacy URL params are present.
func GetFlashMessages(w http.ResponseWriter, r *http.Request) (errorMsg, successMsg string) {
	// 1. Check Cookies
	if cookie, err := r.Cookie(FlashErrorCookie); err == nil && cookie.Value != "" {
		if unescaped, err := url.QueryUnescape(cookie.Value); err == nil {
			errorMsg = unescaped
		} else {
			errorMsg = cookie.Value
		}
		// Clear cookie immediately
		http.SetCookie(w, &http.Cookie{
			Name:     FlashErrorCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	if cookie, err := r.Cookie(FlashSuccessCookie); err == nil && cookie.Value != "" {
		if unescaped, err := url.QueryUnescape(cookie.Value); err == nil {
			successMsg = unescaped
		} else {
			successMsg = cookie.Value
		}
		// Clear cookie immediately
		http.SetCookie(w, &http.Cookie{
			Name:     FlashSuccessCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	// 2. Fallback to query params if not found in cookies
	if errorMsg == "" {
		if qErr := r.URL.Query().Get("error"); qErr != "" {
			errorMsg = qErr
		} else if qImportErr := r.URL.Query().Get("import_error"); qImportErr != "" {
			errorMsg = qImportErr
		}
	}

	if successMsg == "" {
		if qSucc := r.URL.Query().Get("success"); qSucc != "" {
			successMsg = qSucc
		} else if qImportSucc := r.URL.Query().Get("import_success"); qImportSucc != "" {
			successMsg = qImportSucc
		}
	}

	return errorMsg, successMsg
}

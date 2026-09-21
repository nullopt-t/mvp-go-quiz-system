package presenter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityMiddleware_Headers(t *testing.T) {
	handler := SecurityMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// 1. Dynamic route
	req := httptest.NewRequest("GET", "/quizzes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %s", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("Cache-Control") != "no-store, no-cache, must-revalidate, private" {
		t.Errorf("unexpected Cache-Control header: %s", rec.Header().Get("Cache-Control"))
	}

	// 2. Static route
	staticReq := httptest.NewRequest("GET", "/static/css/style.css", nil)
	staticRec := httptest.NewRecorder()
	handler.ServeHTTP(staticRec, staticReq)

	if staticRec.Header().Get("Cache-Control") != "public, max-age=86400" {
		t.Errorf("unexpected static Cache-Control header: %s", staticRec.Header().Get("Cache-Control"))
	}
}

func TestIPRateLimiter_Enforcement(t *testing.T) {
	// Limit: 2 req/s, burst 2
	limiter := NewIPRateLimiter(2, 2, 5*time.Second)
	handler := limiter.LimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.100:12345"

	// First 2 requests within burst should pass
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/admin/login", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected request %d to pass, got %d", i+1, rec.Code)
		}
	}

	// 3rd rapid request should be rate limited with 429
	req := httptest.NewRequest("POST", "/admin/login", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 Too Many Requests, got %d", rec.Code)
	}

	// A different IP should still pass
	reqOther := httptest.NewRequest("POST", "/admin/login", nil)
	reqOther.RemoteAddr = "10.0.0.1:54321"
	recOther := httptest.NewRecorder()
	handler.ServeHTTP(recOther, reqOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("expected other IP request to pass, got %d", recOther.Code)
	}
}

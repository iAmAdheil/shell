package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/router"
)

func TestHealthReportsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	router.New(router.Deps{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Body.String(), `{"status":"ok"}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

package deps

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>gripmock</title>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}

	handler := spaHandler(assets)

	cases := []struct {
		name     string
		path     string
		wantCode int
		wantBody string
	}{
		{"root serves index", "/", http.StatusOK, "gripmock"},
		{"index.html canonicalized to root", "/index.html", http.StatusMovedPermanently, ""},
		{"existing asset", "/assets/app.js", http.StatusOK, "console.log"},
		{"client route falls back to index", "/stubs", http.StatusOK, "gripmock"},
		{"nested client route falls back", "/stubs/create", http.StatusOK, "gripmock"},
		{"missing asset is 404", "/assets/missing.js", http.StatusNotFound, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("path %q: got status %d, want %d", tc.path, rec.Code, tc.wantCode)
			}

			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("path %q: body %q does not contain %q", tc.path, rec.Body.String(), tc.wantBody)
			}
		})
	}
}

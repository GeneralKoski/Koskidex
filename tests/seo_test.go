package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRobotsListsSitemaps(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create an index so robots.txt has a sitemap to list.
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name":"movies"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", "/robots.txt", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Disallow: /indexes/*/search") {
		t.Errorf("robots.txt missing disallow rule: %s", body)
	}
	if !strings.Contains(body, "Sitemap: ") || !strings.Contains(body, "/indexes/movies/sitemap.xml") {
		t.Errorf("robots.txt missing sitemap directive: %s", body)
	}
}

func TestSitemapEscapesXML(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create index and configure sitemap base_url + url field.
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name":"shop"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	settings := `{"sitemap":{"base_url":"https://shop.example","url_field":"url"}}`
	req = httptest.NewRequest("PUT", "/indexes/shop/settings", bytes.NewBufferString(settings))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	// Document whose URL contains characters that must be XML-escaped.
	docs := `[{"id":"1","url":"/p?a=1&b=2"}]`
	req = httptest.NewRequest("POST", "/indexes/shop/documents", bytes.NewBufferString(docs))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", "/indexes/shop/sitemap.xml", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "https://shop.example/p?a=1&amp;b=2") {
		t.Errorf("sitemap did not XML-escape the URL: %s", body)
	}
	if strings.Contains(body, "a=1&b=2") {
		t.Errorf("sitemap contains raw unescaped ampersand: %s", body)
	}
	if !strings.Contains(body, "<lastmod>") {
		t.Errorf("sitemap missing lastmod: %s", body)
	}
}

func TestListDocumentsPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name":"books"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	docs := `[{"id":"a"},{"id":"b"},{"id":"c"}]`
	req = httptest.NewRequest("POST", "/indexes/books/documents", bytes.NewBufferString(docs))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", "/indexes/books/documents?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"total":3`) {
		t.Errorf("expected total 3, got: %s", body)
	}
}

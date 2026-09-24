package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateAndGet(t *testing.T) {
	h := NewServer(NewStore())

	rec := do(t, h, http.MethodPost, "/deployments", `{"name":"web","image":"nginx:1.27"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", rec.Code, rec.Body)
	}
	var created Deployment
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID != "1" || created.Replicas != 1 || created.Status != "pending" {
		t.Fatalf("unexpected deployment: %+v", created)
	}
	if loc := rec.Header().Get("Location"); loc != "/deployments/1" {
		t.Fatalf("Location = %q", loc)
	}

	rec = do(t, h, http.MethodGet, "/deployments/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	var got Deployment
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestList(t *testing.T) {
	h := NewServer(NewStore())

	rec := do(t, h, http.MethodGet, "/deployments", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list: status = %d, body = %s", rec.Code, rec.Body)
	}

	do(t, h, http.MethodPost, "/deployments", `{"name":"a","image":"x"}`)
	do(t, h, http.MethodPost, "/deployments", `{"name":"b","image":"y","replicas":3}`)

	rec = do(t, h, http.MethodGet, "/deployments", "")
	var list []Deployment
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "a" || list[1].Replicas != 3 {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestCreateValidation(t *testing.T) {
	h := NewServer(NewStore())

	cases := map[string]string{
		"malformed":        `{`,
		"missing name":     `{"image":"x"}`,
		"missing image":    `{"name":"x"}`,
		"negative replica": `{"name":"x","image":"y","replicas":-1}`,
		"unknown field":    `{"name":"x","image":"y","color":"red"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/deployments", body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestGetNotFound(t *testing.T) {
	h := NewServer(NewStore())
	if rec := do(t, h, http.MethodGet, "/deployments/42", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := NewServer(NewStore())
	if rec := do(t, h, http.MethodDelete, "/deployments", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

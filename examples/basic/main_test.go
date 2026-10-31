package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestStoreUpdate(t *testing.T) {
	s := newStore()
	if err := s.update(1, "Updated Name", 40, "updated@example.com"); err != nil {
		t.Fatalf("update returned error: %v", err)
	}

	person, err := s.find(1)
	if err != nil {
		t.Fatalf("find returned error: %v", err)
	}
	if person.Name != "Updated Name" || person.Age != 40 || person.Email != "updated@example.com" {
		t.Fatalf("unexpected person after update: %+v", person)
	}
	if err := s.update(999, "Nobody", 1, "nobody@example.com"); err != errPersonNotFound {
		t.Fatalf("update missing person error = %v, want %v", err, errPersonNotFound)
	}
}

func TestEditGetRendersFormAndPreservesState(t *testing.T) {
	recorder := httptest.NewRecorder()
	newMux(newStore()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/people/1/edit?sort=age&dir=desc&page=2&page_size=10&search=alice", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`id="table-form"`,
		`value="Alice Johnson"`,
		`name="sort" value="age"`,
		`name="dir" value="desc"`,
		`name="page" value="2"`,
		`name="search" value="alice"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET body does not contain %q", want)
		}
	}
}

func TestEditPostValidationRetainsValues(t *testing.T) {
	form := url.Values{
		"name":      {""},
		"age":       {"121"},
		"email":     {""},
		"sort":      {"name"},
		"dir":       {"asc"},
		"page":      {"3"},
		"page_size": {"5"},
		"search":    {"bob"},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/people/2/edit", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newMux(newStore()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"Name is required.",
		"Age must be a whole number from 1 to 120.",
		"Email is required.",
		`name="page" value="3"`,
		`name="search" value="bob"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("validation body does not contain %q", want)
		}
	}
}

func TestEditPostUpdatesAndReturnsTable(t *testing.T) {
	form := url.Values{
		"name":      {"Robert Smith"},
		"age":       {"46"},
		"email":     {"robert@example.com"},
		"sort":      {"name"},
		"dir":       {"asc"},
		"page":      {"1"},
		"page_size": {"5"},
	}
	store := newStore()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/people/2/edit", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newMux(store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `id="table-form"`) || !strings.Contains(body, `<table`) {
		t.Fatalf("successful POST did not return updated table fragment")
	}
	person, err := store.find(2)
	if err != nil {
		t.Fatalf("find returned error: %v", err)
	}
	if person.Name != "Robert Smith" || person.Age != 46 || person.Email != "robert@example.com" {
		t.Fatalf("unexpected updated person: %+v", person)
	}
}

func TestEditMissingPerson(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			var body io.Reader
			if method == http.MethodPost {
				body = strings.NewReader(url.Values{"name": {"Nobody"}, "age": {"20"}, "email": {"nobody@example.com"}}.Encode())
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/people/999/edit", body)
			if method == http.MethodPost {
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			newMux(newStore()).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
		})
	}
}

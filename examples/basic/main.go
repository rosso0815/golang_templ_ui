package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/pfistera/golang_templ_ui/table"
)

// Person is the demo row type rendered by the table.
type Person struct {
	ID    int
	Name  string
	Age   int
	Email string
}

// store is a trivial in-memory data source guarded by a mutex; a real
// application would replace this with a database-backed repository.
type store struct {
	mu     sync.Mutex
	people []Person
	nextID int
}

var errPersonNotFound = errors.New("person not found")

func newStore() *store {
	s := &store{nextID: 1}
	seed := []struct {
		Name  string
		Age   int
		Email string
	}{
		{"Alice Johnson", 34, "alice@example.com"},
		{"Bob Smith", 45, "bob@example.com"},
		{"Carol Davis", 29, "carol@example.com"},
		{"David Lee", 52, "david@example.com"},
		{"Eve Martinez", 41, "eve@example.com"},
		{"Frank Wilson", 38, "frank@example.com"},
		{"Grace Kim", 27, "grace@example.com"},
		{"Henry Brown", 60, "henry@example.com"},
		{"Ivy Chen", 33, "ivy@example.com"},
		{"Jack Taylor", 49, "jack@example.com"},
		{"Karen White", 31, "karen@example.com"},
		{"Leo Anderson", 44, "leo@example.com"},
	}
	for _, p := range seed {
		s.people = append(s.people, Person{ID: s.nextID, Name: p.Name, Age: p.Age, Email: p.Email})
		s.nextID++
	}
	return s
}

func (s *store) all() []Person {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Person, len(s.people))
	copy(out, s.people)
	return out
}

func (s *store) find(id int) (Person, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.people {
		if p.ID == id {
			return p, nil
		}
	}
	return Person{}, errPersonNotFound
}

func (s *store) update(id int, name string, age int, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.people {
		if s.people[i].ID == id {
			s.people[i].Name = name
			s.people[i].Age = age
			s.people[i].Email = email
			return nil
		}
	}
	return errPersonNotFound
}

func (s *store) delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.people {
		if p.ID == id {
			s.people = append(s.people[:i], s.people[i+1:]...)
			return nil
		}
	}
	return errPersonNotFound
}

type editFormModel struct {
	ID       int
	Name     string
	Age      string
	Email    string
	NameErr  string
	AgeErr   string
	EmailErr string
	State    table.State
}

func newEditFormModel(p Person, state table.State) editFormModel {
	return editFormModel{
		ID:    p.ID,
		Name:  p.Name,
		Age:   strconv.Itoa(p.Age),
		Email: p.Email,
		State: state,
	}
}

func parseEditForm(r *http.Request, id int, state table.State) editFormModel {
	return editFormModel{
		ID:    id,
		Name:  strings.TrimSpace(r.FormValue("name")),
		Age:   strings.TrimSpace(r.FormValue("age")),
		Email: strings.TrimSpace(r.FormValue("email")),
		State: state,
	}
}

func (m *editFormModel) validate() (int, bool) {
	valid := true
	if m.Name == "" {
		m.NameErr = "Name is required."
		valid = false
	}

	age, err := strconv.Atoi(m.Age)
	if err != nil || age < 1 || age > 120 {
		m.AgeErr = "Age must be a whole number from 1 to 120."
		valid = false
	}

	if m.Email == "" {
		m.EmailErr = "Email is required."
		valid = false
	}
	return age, valid
}

func personColumns() []table.Column[Person] {
	return []table.Column[Person]{
		{
			Header: "Name", SortKey: "name", Sortable: true, Searchable: true,
			Text:    func(p Person) string { return p.Name },
			Compare: func(a, b Person) int { return strings.Compare(a.Name, b.Name) },
		},
		{
			Header: "Age", SortKey: "age", Sortable: true,
			Text:    func(p Person) string { return strconv.Itoa(p.Age) },
			Compare: func(a, b Person) int { return a.Age - b.Age },
		},
		{
			Header: "Email", Searchable: true,
			Text: func(p Person) string { return p.Email },
		},
		{
			Header: "Actions",
			Cell:   func(p Person) templ.Component { return actionButtons(p.ID) },
		},
	}
}

const tableEndpoint = "/people/table"

func buildFragment(r *http.Request, s *store) templ.Component {
	state := table.ParseState(r, 5)
	columns := personColumns()

	items := table.Filter(s.all(), columns, state)
	items = table.Sort(items, columns, state)
	page, total, totalPages := table.Paginate(items, state)

	vm := table.BuildViewModel(page, columns, state, total, totalPages)
	return table.Fragment(vm, tableEndpoint)
}

func newMux(s *store) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		_ = layout("People").Render(templ.WithChildren(r.Context(), buildFragment(r, s)), w)
	})

	mux.HandleFunc("GET "+tableEndpoint, func(w http.ResponseWriter, r *http.Request) {
		_ = buildFragment(r, s).Render(r.Context(), w)
	})

	mux.HandleFunc("DELETE /people/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := s.delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = buildFragment(r, s).Render(r.Context(), w)
	})

	mux.HandleFunc("GET /people/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		person, err := s.find(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		state := table.ParseState(r, 5)
		_ = editForm(newEditFormModel(person, state), tableEndpoint).Render(r.Context(), w)
	})

	mux.HandleFunc("POST /people/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if _, err := s.find(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		state := table.ParseState(r, 5)
		form := parseEditForm(r, id, state)
		age, valid := form.validate()
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			_ = editForm(form, tableEndpoint).Render(r.Context(), w)
			return
		}
		if err := s.update(id, form.Name, age, form.Email); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = buildFragment(r, s).Render(r.Context(), w)
	})

	return mux
}

func main() {
	s := newStore()
	mux := newMux(s)

	addr := ":8080"
	fmt.Printf("listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println("server error:", err)
	}
}

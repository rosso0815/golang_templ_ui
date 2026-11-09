package main

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/rosso0815/rosso0815-go-ui-templ/table"
)

type person struct {
	Name       string
	Email      string
	Role       string
	Department string
}

var people = []person{
	{Name: "Ava Patel", Email: "ava.patel@example.com", Role: "Product Designer", Department: "Design"},
	{Name: "Noah Kim", Email: "noah.kim@example.com", Role: "Backend Engineer", Department: "Engineering"},
	{Name: "Mia Garcia", Email: "mia.garcia@example.com", Role: "Product Manager", Department: "Product"},
	{Name: "Ethan Brown", Email: "ethan.brown@example.com", Role: "Frontend Engineer", Department: "Engineering"},
	{Name: "Sofia Chen", Email: "sofia.chen@example.com", Role: "Customer Success", Department: "Operations"},
	{Name: "Liam Wilson", Email: "liam.wilson@example.com", Role: "Data Analyst", Department: "Analytics"},
	{Name: "Isabella Davis", Email: "isabella.davis@example.com", Role: "Content Designer", Department: "Design"},
	{Name: "Oliver Martin", Email: "oliver.martin@example.com", Role: "Site Reliability Engineer", Department: "Engineering"},
	{Name: "Amara Johnson", Email: "amara.johnson@example.com", Role: "Recruiter", Department: "People"},
	{Name: "Leo Thompson", Email: "leo.thompson@example.com", Role: "Account Executive", Department: "Sales"},
	{Name: "Nina Fischer", Email: "nina.fischer@example.com", Role: "Finance Analyst", Department: "Finance"},
	{Name: "Mateo Lopez", Email: "mateo.lopez@example.com", Role: "Researcher", Department: "Product"},
}

var columns = []table.Column[person]{
	{
		Header: "Name", SortKey: "name", Sortable: true, Searchable: true,
		Text: func(p person) string { return p.Name + " " + p.Email },
		Cell: func(p person) templ.Component {
			return personName(p)
		},
		Compare: func(a, b person) int { return strings.Compare(a.Name, b.Name) },
	},
	{
		Header: "Role", SortKey: "role", Sortable: true, Searchable: true,
		Text:    func(p person) string { return p.Role },
		Compare: func(a, b person) int { return strings.Compare(a.Role, b.Role) },
	},
	{
		Header: "Department", SortKey: "department", Sortable: true, Searchable: true,
		Text:    func(p person) string { return p.Department },
		Compare: func(a, b person) int { return strings.Compare(a.Department, b.Department) },
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handlePage)
	mux.HandleFunc("GET /people/table", handleTable)

	log.Println("table demo listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handlePage(w http.ResponseWriter, r *http.Request) {
	vm := buildViewModel(r)
	render(w, r, Demo(vm))
}

func handleTable(w http.ResponseWriter, r *http.Request) {
	vm := buildViewModel(r)
	render(w, r, table.Fragment(vm, "/people/table"))
}

func buildViewModel(r *http.Request) table.TableViewModel {
	state := table.ParseState(r, 5)
	filtered := table.Filter(people, columns, state)
	sorted := table.Sort(filtered, columns, state)
	page, total, totalPages := table.Paginate(sorted, state)
	return table.BuildViewModel(page, columns, state, total, totalPages)
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		log.Printf("render response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(body.Bytes()); err != nil {
		log.Printf("write response: %v", err)
	}
}

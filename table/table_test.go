package table

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type person struct {
	Name string
	Age  int
}

func personColumns() []Column[person] {
	return []Column[person]{
		{
			Header: "Name", SortKey: "name", Sortable: true, Searchable: true,
			Text:    func(p person) string { return p.Name },
			Compare: func(a, b person) int { return strings.Compare(a.Name, b.Name) },
		},
		{
			Header: "Age", SortKey: "age", Sortable: true,
			Compare: func(a, b person) int { return a.Age - b.Age },
		},
	}
}

func TestSort(t *testing.T) {
	people := []person{{"Charlie", 40}, {"Alice", 30}, {"Bob", 20}}
	cols := personColumns()

	asc := Sort(people, cols, State{SortKey: "name", SortDir: SortAsc})
	if asc[0].Name != "Alice" || asc[2].Name != "Charlie" {
		t.Fatalf("expected alphabetical order, got %+v", asc)
	}

	desc := Sort(people, cols, State{SortKey: "age", SortDir: SortDesc})
	if desc[0].Age != 40 || desc[2].Age != 20 {
		t.Fatalf("expected descending age order, got %+v", desc)
	}

	unsorted := Sort(people, cols, State{SortKey: "unknown"})
	if len(unsorted) != len(people) || unsorted[0].Name != "Charlie" {
		t.Fatalf("expected original order for unknown sort key, got %+v", unsorted)
	}
}

func TestFilter(t *testing.T) {
	people := []person{{"Charlie", 40}, {"Alice", 30}, {"Bob", 20}}
	cols := personColumns()

	filtered := Filter(people, cols, State{Search: "ali"})
	if len(filtered) != 1 || filtered[0].Name != "Alice" {
		t.Fatalf("expected only Alice, got %+v", filtered)
	}

	all := Filter(people, cols, State{Search: ""})
	if len(all) != len(people) {
		t.Fatalf("expected all items for empty search, got %+v", all)
	}

	none := Filter(people, cols, State{Search: "zzz"})
	if len(none) != 0 {
		t.Fatalf("expected no matches, got %+v", none)
	}
}

func TestPaginate(t *testing.T) {
	people := []person{{"A", 1}, {"B", 2}, {"C", 3}, {"D", 4}, {"E", 5}}

	page, total, totalPages := Paginate(people, State{Page: 2, PageSize: 2})
	if total != 5 || totalPages != 3 || len(page) != 2 || page[0].Name != "C" {
		t.Fatalf("unexpected pagination result: page=%+v total=%d totalPages=%d", page, total, totalPages)
	}

	// Out-of-range page clamps to the last page.
	page, _, totalPages = Paginate(people, State{Page: 99, PageSize: 2})
	if totalPages != 3 || len(page) != 1 || page[0].Name != "E" {
		t.Fatalf("expected clamped last page, got page=%+v totalPages=%d", page, totalPages)
	}

	// Empty input still yields at least one page.
	empty, total, totalPages := Paginate([]person{}, State{Page: 1, PageSize: 10})
	if total != 0 || totalPages != 1 || len(empty) != 0 {
		t.Fatalf("expected empty page with totalPages=1, got %+v total=%d totalPages=%d", empty, total, totalPages)
	}
}

func TestParseState(t *testing.T) {
	form := url.Values{
		"sort": {"name"}, "dir": {"desc"}, "page": {"2"}, "page_size": {"5"}, "search": {"foo"},
	}
	r := &http.Request{Method: http.MethodGet, Form: form}

	state := ParseState(r, 10)
	if state.SortKey != "name" || state.SortDir != SortDesc || state.Page != 2 || state.PageSize != 5 || state.Search != "foo" {
		t.Fatalf("unexpected state: %+v", state)
	}

	// Defaults apply for invalid/missing values.
	r2 := &http.Request{Method: http.MethodGet, Form: url.Values{"dir": {"sideways"}}}
	defaults := ParseState(r2, 10)
	if defaults.Page != 1 || defaults.PageSize != 10 || defaults.SortDir != SortAsc {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
}

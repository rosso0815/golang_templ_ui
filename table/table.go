// Package table provides reusable server-driven data table logic: sorting,
// filtering, and pagination over a generic slice of rows, plus a builder that
// resolves that logic into a plain TableViewModel for rendering with templ.
package table

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Column describes one column of a table over rows of type T.
type Column[T any] struct {
	// Header is the label shown in the column heading.
	Header string
	// SortKey identifies this column in State.SortKey; required if Sortable.
	SortKey string
	// Sortable enables clickable header sorting for this column.
	Sortable bool
	// Compare orders two rows for this column; required if Sortable.
	Compare func(a, b T) int
	// Searchable includes this column's Text value in Filter's substring match.
	Searchable bool
	// Text returns the plain-text value of this column for a row; required if
	// Searchable, and used as the default Cell rendering when Cell is nil.
	Text func(item T) string
	// Cell renders the table cell for a row. Defaults to a text node from Text.
	Cell func(item T) templ.Component
}

// State captures the current sort/filter/pagination request for a table.
type State struct {
	SortKey  string
	SortDir  string
	Page     int
	PageSize int
	Search   string
}

// ParseState reads table state from request form values, applying
// defaultPageSize when the page-size field is absent or invalid.
func ParseState(r *http.Request, defaultPageSize int) State {
	_ = r.ParseForm()

	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.FormValue("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = defaultPageSize
	}

	dir := r.FormValue("dir")
	if dir != SortAsc && dir != SortDesc {
		dir = SortAsc
	}

	return State{
		SortKey:  r.FormValue("sort"),
		SortDir:  dir,
		Page:     page,
		PageSize: pageSize,
		Search:   r.FormValue("search"),
	}
}

// Sort returns a new slice of items ordered by the column matching
// state.SortKey, or the original order if no column matches.
func Sort[T any](items []T, columns []Column[T], state State) []T {
	var col *Column[T]
	for i := range columns {
		if columns[i].Sortable && columns[i].SortKey == state.SortKey {
			col = &columns[i]
			break
		}
	}
	if col == nil {
		return items
	}

	sorted := make([]T, len(items))
	copy(sorted, items)

	sort.SliceStable(sorted, func(i, j int) bool {
		cmp := col.Compare(sorted[i], sorted[j])
		if state.SortDir == SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
	return sorted
}

// Filter returns only the items whose searchable column text contains
// state.Search (case-insensitive). Returns items unchanged if Search is empty.
func Filter[T any](items []T, columns []Column[T], state State) []T {
	search := strings.TrimSpace(state.Search)
	if search == "" {
		return items
	}
	needle := strings.ToLower(search)

	var searchableCols []Column[T]
	for _, c := range columns {
		if c.Searchable && c.Text != nil {
			searchableCols = append(searchableCols, c)
		}
	}
	if len(searchableCols) == 0 {
		return items
	}

	filtered := make([]T, 0, len(items))
	for _, item := range items {
		for _, c := range searchableCols {
			if strings.Contains(strings.ToLower(c.Text(item)), needle) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// Paginate slices items to the page described by state, clamping to the
// valid range, and reports the total item/page counts for the full slice.
func Paginate[T any](items []T, state State) (page []T, totalItems, totalPages int) {
	totalItems = len(items)
	pageSize := state.PageSize
	if pageSize < 1 {
		pageSize = 1
	}

	totalPages = (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	current := state.Page
	if current < 1 {
		current = 1
	}
	if current > totalPages {
		current = totalPages
	}

	start := (current - 1) * pageSize
	if start > totalItems {
		start = totalItems
	}
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	return items[start:end], totalItems, totalPages
}

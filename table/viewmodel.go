package table

import (
	"strconv"

	"github.com/a-h/templ"
)

// toStr is a small helper for embedding ints in templ attribute values.
func toStr(n int) string {
	return strconv.Itoa(n)
}

// HeaderCell is a single resolved column header for TableViewModel.
type HeaderCell struct {
	Label    string
	SortKey  string
	Sortable bool
	// Active is true when the table is currently sorted by this column.
	Active bool
	// Dir is the sort direction this header link would apply next ("asc"/"desc").
	Dir string
}

// TableViewModel is the plain, non-generic data the templ components render.
type TableViewModel struct {
	FormID     string
	Headers    []HeaderCell
	Rows       [][]templ.Component
	State      State
	TotalItems int
	TotalPages int
}

// BuildViewModel resolves columns, rows and state into a TableViewModel ready
// for rendering. Callers typically pass an already sorted/filtered/paginated
// page of items (see Sort, Filter, Paginate).
func BuildViewModel[T any](items []T, columns []Column[T], state State, totalItems, totalPages int) TableViewModel {
	headers := make([]HeaderCell, len(columns))
	for i, c := range columns {
		h := HeaderCell{
			Label:    c.Header,
			SortKey:  c.SortKey,
			Sortable: c.Sortable,
		}
		if c.Sortable && c.SortKey == state.SortKey {
			h.Active = true
			if state.SortDir == SortDesc {
				h.Dir = SortAsc
			} else {
				h.Dir = SortDesc
			}
		} else {
			h.Dir = SortAsc
		}
		headers[i] = h
	}

	rows := make([][]templ.Component, len(items))
	for r, item := range items {
		cells := make([]templ.Component, len(columns))
		for c, col := range columns {
			switch {
			case col.Cell != nil:
				cells[c] = col.Cell(item)
			case col.Text != nil:
				cells[c] = templ.Raw(templ.EscapeString(col.Text(item)))
			default:
				cells[c] = templ.NopComponent
			}
		}
		rows[r] = cells
	}

	return TableViewModel{
		FormID:     "table-form",
		Headers:    headers,
		Rows:       rows,
		State:      state,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

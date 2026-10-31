# golang_templ_ui — Table Component Specification

Module: `github.com/pfistera/golang_templ_ui`
Package: `table`

## 1. Purpose

A reusable Go library for rendering server-driven data tables with `templ` +
Bootstrap 5, interactive via HTMX (sorting, pagination, free-text search, and
per-row actions), without requiring a client-side JS framework.

## 2. Non-goals

- No client-side-only (JS-driven) sorting/filtering mode.
- No CSS bundling — consumers supply Bootstrap 5 themselves (CDN or asset pipeline).
- No persistence/ORM layer — the library operates on in-memory `[]T` slices per request; data fetching is the caller's responsibility.
- No built-in auth/CSRF handling on row-action routes (edit/delete) — left to the host application.

## 3. Data model (`table` package, generic Go layer)

### 3.1 `Column[T any]`

| Field        | Type                          | Required when        | Meaning |
|--------------|-------------------------------|-----------------------|---------|
| `Header`     | `string`                      | always                | Column heading label |
| `SortKey`    | `string`                      | `Sortable == true`    | Value matched against `State.SortKey` |
| `Sortable`   | `bool`                        | —                     | Enables clickable sort header |
| `Compare`    | `func(a, b T) int`            | `Sortable == true`    | `<0` if `a<b`, `0` if equal, `>0` if `a>b` |
| `Searchable` | `bool`                        | —                     | Include column in free-text search |
| `Text`       | `func(item T) string`         | `Searchable == true`  | Plain-text value; also default cell rendering when `Cell` is nil |
| `Cell`       | `func(item T) templ.Component`| —                     | Custom cell renderer (e.g. row action buttons); overrides `Text` |

Constraints not enforced at compile time (caller responsibility):
- A `Sortable` column without `Compare` will panic when that sort key is selected (`Sort` dereferences `Compare`).
- A `Searchable` column without `Text` is silently skipped by `Filter`.
- A column with neither `Cell` nor `Text` renders an empty cell (`templ.NopComponent`).

### 3.2 `State`

```go
type State struct {
    SortKey  string
    SortDir  string // SortAsc ("asc") or SortDesc ("desc")
    Page     int    // 1-based
    PageSize int
    Search   string
}
```

### 3.3 `ParseState(r *http.Request, defaultPageSize int) State`

Reads `sort`, `dir`, `page`, `page_size`, `search` from the request's parsed
form values (`r.ParseForm()`, so both query string and `application/x-www-form-urlencoded` bodies work).

Normalization rules:
- `page`: parsed as int; any parse error or value `< 1` → `1`.
- `page_size`: parsed as int; any parse error or value `< 1` → `defaultPageSize`.
- `dir`: any value other than exactly `"asc"` or `"desc"` → `"asc"`.
- `sort`, `search`: passed through verbatim (including empty string).

## 4. Query pipeline

Callers compose three pure functions in this fixed order — **filter, then
sort, then paginate** — followed by view-model construction:

```go
items := table.Filter(all, columns, state)
items  = table.Sort(items, columns, state)
page, total, totalPages := table.Paginate(items, state)
vm := table.BuildViewModel(page, columns, state, total, totalPages)
```

### 4.1 `Filter[T](items []T, columns []Column[T], state State) []T`

- No-op (returns `items` unchanged, same backing slice) when `strings.TrimSpace(state.Search) == ""`.
- Otherwise: case-insensitive substring match. An item is kept if **any**
  `Searchable` column's `Text(item)` contains the (lowercased, trimmed) search
  term. Columns without `Text` are ignored for search purposes.
- If no column is `Searchable`, returns `items` unchanged.

### 4.2 `Sort[T](items []T, columns []Column[T], state State) []T`

- Returns a new slice (input is never mutated) ordered by the first `Sortable`
  column whose `SortKey == state.SortKey`.
- If no column matches (including `state.SortKey == ""`), returns the
  original slice unmodified (original order preserved).
- Uses `sort.SliceStable` — equal elements (per `Compare`) retain relative
  input order.
- `state.SortDir == SortDesc` reverses comparison result; anything else
  (including empty) behaves as ascending.

### 4.3 `Paginate[T](items []T, state State) (page []T, totalItems, totalPages int)`

- `totalItems = len(items)`.
- `pageSize` is clamped to `>= 1` (a `state.PageSize < 1` is treated as `1` to avoid divide-by-zero).
- `totalPages = ceil(totalItems / pageSize)`, clamped to `>= 1` (an empty
  result set still reports 1 page).
- `state.Page` is clamped into `[1, totalPages]` — out-of-range pages
  (including page 0 or huge page numbers) resolve to the nearest valid page,
  they never error or return an empty slice due to overshoot.
- Returns a sub-slice `items[start:end]`; this is a slice of the input
  backing array, not a copy.

### 4.4 `BuildViewModel[T](items []T, columns []Column[T], state State, totalItems, totalPages int) TableViewModel`

Resolves columns + the (already filtered/sorted/paginated) `items` page into
a **plain, non-generic** `TableViewModel`:

```go
type HeaderCell struct {
    Label    string
    SortKey  string
    Sortable bool
    Active   bool   // true if this column is the current sort key
    Dir      string // the direction this header's link will apply NEXT
}

type TableViewModel struct {
    FormID     string // constant "table-form"
    Headers    []HeaderCell
    Rows       [][]templ.Component // Rows[i][j] = cell for row i, column j
    State      State
    TotalItems int
    TotalPages int
}
```

- `HeaderCell.Dir` semantics: if the column is currently active and sorted
  ascending, `Dir = "desc"` (clicking again reverses); if active and
  descending, `Dir = "asc"`; if inactive, `Dir = "asc"` (first click sorts
  ascending).
- Cell resolution priority per column: `Cell` if set, else
  `templ.Raw(templ.EscapeString(Text(item)))` if `Text` is set, else
  `templ.NopComponent` (empty cell). Text is HTML-escaped before being
  wrapped in `Raw`, so it is safe against injection.
- `items` passed in is expected to already be the *current page* (i.e. call
  `Paginate` first) — `BuildViewModel` does not paginate.

## 5. Rendering layer (`table.templ` — generated into `table_templ.go`)

### 5.1 `Fragment(vm TableViewModel, endpoint string) templ.Component`

The single entry point most consumers render. Produces one `<form id="table-form">`
containing, in order:
1. Hidden inputs: `sort`, `dir`, `page`, `page_size` (mirroring `vm.State`; `search` is carried by the visible search input, not a hidden field).
2. The search box (`<input name="search">`).
3. The `<table>` (see 5.2).
4. Pagination controls (see 5.3).

### 5.2 `Table(vm TableViewModel, endpoint string) templ.Component`

Renders `<table class="table table-striped table-hover align-middle">`:
- Header row: one `<th>` per `vm.Headers` entry. Sortable headers render an
  `<a>` with `hx-get`, `hx-include="#table-form"`, `hx-vals` (JSON: new
  `sort`, `dir`, and reset `page` to `"1"`), `hx-target="#table-form"`,
  `hx-swap="outerHTML"`. An arrow (`&darr;`/`&uarr;`) is appended when
  `Active`.
- Body: one `<tr>` per `vm.Rows` entry, one `<td>` per cell. If `vm.Rows` is
  empty, renders a single full-width "No results" row (`colspan = len(Headers)`).

### 5.3 `Pagination(vm TableViewModel, endpoint string) templ.Component`

Renders a Bootstrap `<nav><ul class="pagination">` with:
- A "Previous" item (`page = vm.State.Page - 1`), `disabled` class when `Page <= 1`.
- One numbered item per page `1..TotalPages`, `active` class on the current page.
- A "Next" item (`page = vm.State.Page + 1`), `disabled` class when `Page >= TotalPages`.
- All links carry the same `hx-include`/`hx-target`/`hx-swap` contract as headers.

**Note:** `Pagination` always renders a link for every page number (no
truncation/ellipsis for very large `TotalPages`). Acceptable for the
library's target use cases; large datasets should use a smaller `PageSize`
or a future truncated variant.

## 6. HTMX / HTTP contract (consumer responsibility)

A host handler must:
1. Accept `GET` requests at some `endpoint` string (passed to `Fragment`/`Table`/`Pagination` verbatim — the library does not validate or register routes).
2. Call `ParseState(r, defaultPageSize)` → `Filter` → `Sort` → `Paginate` → `BuildViewModel`, in that order.
3. Render `table.Fragment(vm, endpoint)` for that route (fragment-only response — no `<html>`/layout wrapper).
4. Render the same fragment (or `Fragment` wrapped in a full-page layout) for the initial full-page `GET` route.
5. Optionally register additional row-action routes (e.g. `DELETE /items/{id}`) that mutate the data store and then re-render `buildFragment` so the swapped `#table-form` reflects the change — action buttons must target `#table-form` / `outerHTML` to stay consistent with sort/search/pagination.

Because everything (search, sort, pagination, and, by convention, row
actions) targets and replaces the same `#table-form` element via
`hx-swap="outerHTML"`, state and displayed result counts never go out of
sync with each other.

## 7. Extension points

- **Custom cell rendering**: set `Column.Cell` to render arbitrary `templ.Component` (badges, links, action buttons, icons).
- **Custom compare/search logic**: `Compare` and `Text` are arbitrary functions — multi-field sort/search is possible by combining fields inside the closure.
- **Custom endpoint routing**: `endpoint` is just a string passed to `hx-get`; the library has no opinion on URL structure.
- **Page size**: caller-controlled via `ParseState`'s `defaultPageSize` argument and the `page_size` hidden field (present in the form but not exposed as a user control by default — a consumer could add a page-size `<select>` that submits into the same form).

## 8. Testing strategy

- `table/table_test.go`: table-driven unit tests over `Sort`, `Filter`, `Paginate`, `ParseState` covering: ascending/descending sort, unknown sort key (no-op), search match/no-match/empty-search, page clamping (both directions), empty input set, and invalid/missing form values.
- Rendering (`.templ` output) is exercised indirectly via `examples/basic`, which is smoke-tested by running the server and curling `/`, the fragment endpoint with sort/page params, and a search query.
- No golden-file/snapshot tests on generated HTML in this version — a future addition if visual regressions become a concern.

## 9. Known limitations / risks

1. `Sort` calling a `nil` `Compare` on a `Sortable` column panics — not currently guarded; consider validating `Column` slices at startup in a future revision.
2. `Pagination` renders unbounded page-number links — no windowing for very large `TotalPages`.
3. Row-action routes (edit/delete) are entirely a consumer concern; the library provides no CSRF/auth scaffolding for state-changing HTMX requests (`hx-delete`, etc.) — hosts must add their own protections.
4. `templ`'s native generics support is intentionally not used for the component API (`Table`, `Fragment`, etc. take a non-generic `TableViewModel`) to avoid depending on that still-evolving feature.

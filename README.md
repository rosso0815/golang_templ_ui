# golang_templ_ui

A reusable Go library for rendering server-driven, sortable/paginated/searchable data tables using [templ](https://templ.guide) + [Bootstrap 5](https://getbootstrap.com/), interactive via [HTMX](https://htmx.org).

## Install

```sh
go get github.com/pfistera/golang_templ_ui
```

You'll also need the `templ` CLI to generate code from `.templ` files:

```sh
go install github.com/a-h/templ/cmd/templ@latest
```

## Usage

1. Define columns for your row type:

```go
columns := []table.Column[Person]{
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
        Header: "Actions",
        Cell: func(p Person) templ.Component { return myActionButtons(p.ID) },
    },
}
```

2. In your handler, resolve state → data → view model, and render `table.Fragment`:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    state := table.ParseState(r, 10) // default page size
    items := table.Filter(allPeople, columns, state)
    items = table.Sort(items, columns, state)
    page, total, totalPages := table.Paginate(items, state)

    vm := table.BuildViewModel(page, columns, state, total, totalPages)
    table.Fragment(vm, "/people/table").Render(r.Context(), w)
}
```

`table.Fragment` renders a single `<form>` containing the search box, `<table>`, and pagination controls together. Every sort header, the search input, and every pagination link use `hx-include` to submit the form's current state and `hx-target`/`hx-swap="outerHTML"` to replace the whole form — so state and result counts always stay in sync. Wire the same handler to a `GET` route and call it for both the initial full-page render and subsequent HTMX fragment requests.

See [`examples/basic`](examples/basic) for a complete runnable app (in-memory data, edit/delete row actions).

The basic example uses an inline HTMX edit form at `GET /people/{id}/edit` and
`POST /people/{id}/edit`. It preserves the table's current search, sort, and
pagination state while validating and updating the in-memory person.

## Development

```sh
make generate   # templ generate ./...
make build      # go build ./...
make test       # go test ./...
make vet        # go vet ./...
make run-example  # run the demo app on :8080
```

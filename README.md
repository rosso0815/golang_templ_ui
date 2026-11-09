# golang_templ_ui

A reusable Go library for rendering server-driven, sortable/paginated/searchable data tables using [templ](https://templ.guide) + [Bootstrap 5](https://getbootstrap.com/), interactive via [HTMX](https://htmx.org).

## Install

```sh
go get github.com/rosso0815/rosso0815-go-ui-templ
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

See [`examples/basic`](examples/basic) for a complete runnable app with in-memory sample data and
HTMX-powered search, sorting, and pagination. Start it with:

```sh
make run-example
```

Then open http://localhost:8080.

The demo keeps its data in memory and shows how to render the same table
fragment inside a full page and for subsequent HTMX requests.

## Development

```sh
make generate   # templ generate ./...
make build      # go build ./...
make test       # go test ./...
make vet        # go vet ./...
make run-example  # run the demo app on :8080
```

## CI and releases

GitHub Actions runs on pushes and pull requests using the Go version in
`go.mod`. It installs the templ CLI at the module's templ version, generates
the Go code, and runs `go build ./...` and `go test ./...`.

To publish a release, push a semantic-version tag such as `v1.2.3`:

```sh
git tag v1.2.3
git push origin v1.2.3
```

After CI succeeds, the workflow creates a GitHub Release with generated
release notes. Tags with a prerelease suffix (for example `v1.2.3-rc.1`)
create prereleases. GitHub automatically supplies source archives; this library
does not publish binary artifacts. The repository's Actions settings must
allow the workflow's `GITHUB_TOKEN` to have `contents: write` permission for
the release job.

# apcupsd

Package `apcupsd` provides a client for the [apcupsd](http://www.apcupsd.org/)
Network Information Server (NIS).  MIT Licensed.

## Development

### Prerequisites

- [Go 1.26](https://go.dev/dl/) or later

### Getting the code

```bash
git clone https://github.com/acaylor/apcupsd.git
cd apcupsd
go mod download
```

### Build

```bash
go build ./...
```

### Test

Run the test suite:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

### Lint and static analysis

```bash
go vet ./...
```

The repository also runs [staticcheck](https://staticcheck.io/) and [enumcheck](https://pkg.go.dev/loov.dev/enumcheck) in CI; install and run them locally if you want to match CI:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
go install loov.dev/enumcheck@latest
staticcheck ./...
enumcheck ./...
```

---

## Credits

This is a community-maintained fork. The original project was created and maintained by [Matt Layher](https://github.com/mdlayher) ([github.com/mdlayher/apcupsd](https://github.com/mdlayher/apcupsd)). Many thanks for the original work.

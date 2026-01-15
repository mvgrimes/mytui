APP      := "sqlcli"
VERSION  := `perl -nE'm{VERSION\s*=\s*"(\d+\.\d+.\d+)"} && print $1' ./cmd/main.go`

build:
  echo "Building verions {{VERSION}} of {{APP}}"
  go build -o sqlcli cmd/sqlcli/main.go

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

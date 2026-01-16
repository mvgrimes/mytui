APP      := "mycli-go"
VERSION  := `perl -nE'm{VERSION\s*=\s*"(\d+\.\d+.\d+)"} && print $1' ./cmd/mycli-go/main.go`

build:
  echo "Building verions {{VERSION}} of {{APP}}"
  go build -o mycli-go cmd/mycli-go/main.go

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

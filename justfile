APP      := "mytui"
VERSION  := `perl -nE'm{version\s*=\s*"(\d+\.\d+.\d+)"} && print $1' ./cmd/mytui/main.go`

build:
  echo "Building verions {{VERSION}} of {{APP}}"
  go build -o {{APP}} cmd/mytui/main.go

lint:
  go vet ./... || true
  golangci-lint run ./... || true
  govulncheck ./...

fmt:
  go fmt ./...

test:
  go test ./...

release:
  go mod tidy
  just fmt
  just build
  git diff --exit-code
  git tag "{{VERSION}}"
  git push
  git release
  git push --tags
  goreleaser release --clean

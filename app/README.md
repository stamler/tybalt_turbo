# tybalt

## Serve in development

`go run main.go serve`

## Build For macOS

`GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build`

## Build for x64 linux

`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`

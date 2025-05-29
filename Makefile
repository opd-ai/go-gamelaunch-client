fmt:
	find . -name '*.go' -not -path './vendor/*' -exec gofumpt -extra -s -w {} \;

prompt: fmt
	code2prompt --output prompt.md .

godoc:
	godocdown -o pkg/dgclient/README.md pkg/dgclient
	godocdown -o pkg/tui/README.md pkg/tui
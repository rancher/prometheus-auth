lint:
	pre-commit run --all-files

test:
	go test -race -count=1 -v ./...


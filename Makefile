.PHONY: test gen
test:
	go vet ./... && go test -race -count=1 ./...
gen:
	go generate ./... && go test -run Golden -args -update

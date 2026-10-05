.PHONY: test vet run compose-up
test:
	go test ./...
vet:
	go vet ./...
run:
	go run ./cmd/api
compose-up:
	docker compose up --build

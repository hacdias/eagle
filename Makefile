.PHONY: build ngrok dependencies

build:
	go build -o main ./cmd/eagle

ngrok:
	ngrok http --region eu 8080

dependencies:
	docker compose up

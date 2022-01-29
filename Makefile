.PHONY: build pgadmin postgres ngrok

build:
	go build -o main ./cmd/eagle

postgres:
	docker run \
		-p 5432:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=pgpassword \
		-e POSTGRES_DB=eagle \
		postgres

ngrok:
	ngrok http -region eu 8080
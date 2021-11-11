.PHONY: build pgadmin postgres

build:
	go build -o main ./cmd/eagle

postgres:
	docker run \
		-p 5432:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=pgpassword \
		postgres
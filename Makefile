.PHONY: meilisearch build pgadmin postgres

meilisearch:
	docker run -it --rm -p 7700:7700 getmeili/meilisearch:latest

build:
	go build -o main ./cmd/eagle

postgres:
	docker run \
		-p 5432:5432 \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=pgpassword \
		postgres
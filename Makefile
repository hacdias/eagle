.PHONY: meilisearch build

meilisearch:
	docker run -it --rm -p 7700:7700 getmeili/meilisearch:latest

build:
	go build -o main ./cmd/eagle

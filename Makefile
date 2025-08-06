.PHONY: build pgadmin meilisearch ngrok imgproxy

build:
	go build -o main ./cmd/eagle

meilisearch:
	docker run -it --rm \
		-p 7700:7700 \
		-e MEILI_ENV='development' \
		getmeili/meilisearch:v1.16

ngrok:
	ngrok http --region eu 8080

imgproxy:
	docker run -p 8085:8080 -it \
		-v $(PWD)/testing/imgproxy/:/data/ \
		--env IMGPROXY_LOCAL_FILESYSTEM_ROOT=/data/ \
		--env IMGPROXY_JPEG_PROGRESSIVE=true \
		--env IMGPROXY_AUTO_ROTATE=true \
		--env IMGPROXY_STRIP_METADATA=true \
		--env IMGPROXY_STRIP_COLOR_PROFILE=true \
		--env IMGPROXY_ALLOWED_SOURCES="local://" \
		--env IMGPROXY_BASE_URL="local:///" \
		darthsim/imgproxy

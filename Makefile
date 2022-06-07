.PHONY: build pgadmin postgres ngrok imgproxy

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

imgproxy:
	docker run -p 8085:8080 -it \
		-v $(PWD)/imgproxy/:/data/ \
		--env IMGPROXY_LOCAL_FILESYSTEM_ROOT=/data/ \
		--env IMGPROXY_JPEG_PROGRESSIVE=true \
		--env IMGPROXY_AUTO_ROTATE=true \
		--env IMGPROXY_STRIP_METADATA=true \
		--env IMGPROXY_STRIP_COLOR_PROFILE=true \
		--env IMGPROXY_ALLOWED_SOURCES="local://" \
		--env IMGPROXY_BASE_URL="local:///" \
		darthsim/imgproxy

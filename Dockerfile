FROM golang:1.15-alpine3.12 as build

COPY . /src/
WORKDIR /src/
RUN go build

FROM alpine:3.12

COPY --from=build /src/eagle /bin/
WORKDIR /app

VOLUME /app/source
VOLUME /app/public

EXPOSE 8080
CMD ["eagle"]

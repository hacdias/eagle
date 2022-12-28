FROM golang:1.19-alpine3.16 as build

RUN apk update && \
    apk add --no-cache git gcc g++ musl-dev

WORKDIR /eagle/

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download

COPY . /eagle/
RUN go build -o main ./cmd/eagle

FROM alpine:3.12

COPY --from=build /eagle/main /bin/eagle

ENV UID 501
ENV GID 20

RUN apk update && \
  apk add --no-cache git ca-certificates openssh tzdata mailcap && \
  addgroup -g $UID eagle && \
  adduser --system --uid $UID --ingroup eagle --home /home/eagle eagle && \
  mkdir /app /app/source /app/public /app/activitypub /imgproxy && \
  chown -R eagle:eagle /app /imgproxy

USER eagle

RUN git config --global user.name "Eagle" && \
  git config --global user.email "eagle@eagle"

WORKDIR /app
VOLUME /app/source
VOLUME /app/public
VOLUME /app/activitypub
VOLUME /imgproxy

EXPOSE 8080

CMD ["eagle"]

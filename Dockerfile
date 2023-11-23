FROM golang:1.21-alpine3.18 as build

ENV HUGO_VERSION v0.120.2

RUN apk update && \
    apk add --no-cache git gcc g++ musl-dev && \
    go install github.com/magefile/mage@latest

WORKDIR /hugo
RUN git clone --branch $HUGO_VERSION https://github.com/gohugoio/hugo.git . &&\
  go build -v --tags extended

RUN mage hugo && mage install

WORKDIR /eagle/

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download

COPY . /eagle/
RUN go build -o main ./cmd/eagle

FROM alpine:3.18

COPY --from=build /eagle/main /bin/eagle
COPY --from=build /hugo/hugo /bin/hugo

ENV UID 501
ENV GID 20

RUN apk update && \
  apk add --no-cache git ca-certificates openssh tzdata mailcap && \
  addgroup -g $UID eagle && \
  adduser --system --uid $UID --ingroup eagle --home /home/eagle eagle && \
  mkdir /app /app/source /app/public /app/data /imgproxy && \
  chown -R eagle:eagle /app /imgproxy

USER eagle

RUN git config --global user.name "Eagle" && \
  git config --global user.email "eagle@eagle" && \
  git config --global pull.rebase true

WORKDIR /app
VOLUME /app/source
VOLUME /app/public
VOLUME /app/data
VOLUME /imgproxy

EXPOSE 8080

CMD ["eagle"]

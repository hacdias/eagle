FROM golang:1.15-alpine3.12 as build

ENV HUGO_VERSION v0.76.3

RUN apk update && \
    apk add --no-cache git gcc g++ musl-dev && \
    go get github.com/magefile/mage

WORKDIR /hugo
RUN git clone --branch $HUGO_VERSION https://github.com/gohugoio/hugo.git . &&\
  go build -v --tags extended

RUN mage hugo && mage install

WORKDIR /eagle/
COPY . /eagle/
RUN go build

FROM alpine:3.12

COPY --from=build /eagle/eagle /bin/eagle
COPY --from=build /hugo/hugo /bin/hugo

ENV UID 501
ENV GID 20

RUN apk update && \
  apk add --no-cache git ca-certificates && \
  addgroup -g $UID eagle && \
  adduser --system --uid $UID --ingroup eagle --home /eagle eagle && \
  mkdir /app && \
  chown eagle:eagle /app

USER eagle

RUN git config --global user.name "Eagle" && \
  git config --global user.email "eagle@eagle"

WORKDIR /app
VOLUME /app/source
VOLUME /app/public

EXPOSE 8080
CMD ["eagle"]

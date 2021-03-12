FROM golang:1.16-alpine3.12 as build

ENV HUGO_VERSION v0.80.0

RUN apk update && \
    apk add --no-cache git gcc g++ musl-dev && \
    go get github.com/magefile/mage

WORKDIR /hugo
RUN git clone --branch $HUGO_VERSION https://github.com/gohugoio/hugo.git . &&\
  go build -v --tags extended

RUN mage hugo && mage install

WORKDIR /eagle/
COPY . /eagle/
RUN ./build.sh

FROM alpine:3.12

COPY --from=build /eagle/main /bin/eagle
COPY --from=build /hugo/hugo /bin/hugo

ENV UID 501
ENV GID 20

RUN apk update && \
  apk add --no-cache git ca-certificates openssh && \
  addgroup -g $UID eagle && \
  adduser --system --uid $UID --ingroup eagle --home /home/eagle eagle && \
  mkdir /app /app/source /app/public && \
  chown -R eagle:eagle /app

USER eagle

RUN git config --global user.name "Eagle" && \
  git config --global user.email "eagle@eagle"

WORKDIR /app
VOLUME /app/source
VOLUME /app/public
VOLUME /app/activity

EXPOSE 8080
CMD ["eagle"]

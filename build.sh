#!/usr/bin/env sh

go install github.com/GeertJohan/go.rice/rice
cd server && rice embed-go && cd ..
go build -o main ./cmd/eagle

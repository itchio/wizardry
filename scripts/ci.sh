#!/bin/sh -xe

go version
go test -v -cover -coverprofile=coverage.txt -race ./...
curl -s https://codecov.io/bash | bash


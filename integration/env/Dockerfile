FROM golang:1.24.0-alpine

ENV GO111MODULE=on

WORKDIR /

COPY go.mod go.sum ./
RUN go mod download
COPY . .

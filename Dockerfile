# syntax=docker/dockerfile:1

FROM --platform=linux/amd64 golang:1.22-alpine AS builder

WORKDIR /build
ADD go.mod .
ADD go.sum .
RUN go mod download && go mod verify
COPY . .

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
RUN go build -ldflags "-s -w -X 'github.com/mjvrijn/quotebot/main.Version=`git describe --tags --abbrev=0`'" -o /app/quotebot

FROM --platform=linux/amd64 alpine
WORKDIR /app
COPY --from=builder /app/quotebot .
CMD ["./quotebot"]
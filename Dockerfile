FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/potatobot ./cmd/potatobot

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S bot && adduser -S -G bot bot
COPY --from=build /out/potatobot /usr/local/bin/potatobot
USER bot:bot
ENTRYPOINT ["/usr/local/bin/potatobot"]

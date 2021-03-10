FROM golang:alpine

RUN apk update && apk add git ffmpeg ca-certificates && update-ca-certificates

WORKDIR /bot

COPY . .

RUN CGO_ENABLED=0 go build -o musicbot .

ENTRYPOINT ["/bot/musicbot", "-f", "bot.toml"]

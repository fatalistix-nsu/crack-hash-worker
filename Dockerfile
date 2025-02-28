FROM golang:1.23.5-alpine AS build
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -extldflags '-static'" -o ./app cmd/crack-hash-worker/main.go

RUN apk add upx
RUN upx ./app

FROM scratch AS release
COPY --from=build /build/app /app
COPY .env .env

EXPOSE 6970

ENTRYPOINT ["/app"]

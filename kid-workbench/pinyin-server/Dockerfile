FROM golang:1.26-alpine AS build

WORKDIR /src/pinyin-server
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY pinyin-server/go.mod pinyin-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY pinyin-server/ ./
RUN CGO_ENABLED=0 go build -o /out/pinyin-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=build /out/pinyin-server ./pinyin-server
ENV APP_ADDR=:19111
EXPOSE 19111
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19111/healthz >/dev/null || exit 1
CMD ["./pinyin-server"]

FROM golang:1.26-alpine AS build
WORKDIR /src/math-server
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY math-server/go.mod math-server/go.sum ./
RUN GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct go mod download
COPY shared-go/ /src/shared-go/
COPY math-server/ ./
RUN CGO_ENABLED=0 go build -o /out/math-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/math-server ./math-server
USER app
ENV APP_ADDR=:19141
EXPOSE 19141
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19141/healthz >/dev/null || exit 1
CMD ["./math-server"]

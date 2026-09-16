FROM golang:1.26-alpine AS build
WORKDIR /src/chengyu-server
ENV GOPROXY=https://goproxy.cn,direct
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY chengyu-server/go.mod chengyu-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY chengyu-server/ ./
RUN CGO_ENABLED=0 go build -o /out/chengyu-server ./cmd/server
FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=build /out/chengyu-server ./chengyu-server
ENV APP_ADDR=:19181
EXPOSE 19181
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19181/healthz >/dev/null || exit 1
CMD ["./chengyu-server"]

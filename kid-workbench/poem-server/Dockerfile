FROM golang:1.26-alpine AS build
WORKDIR /src/poem-server
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY poem-server/go.mod poem-server/go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY poem-server/ ./
RUN CGO_ENABLED=0 go build -o /out/poem-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/poem-server ./poem-server
USER app
ENV APP_ADDR=:19161
EXPOSE 19161
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19161/healthz >/dev/null || exit 1
CMD ["./poem-server"]

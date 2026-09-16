FROM golang:1.26-alpine AS build
WORKDIR /src/phrase-server
ENV GOPROXY=https://goproxy.cn,direct
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY phrase-server/go.mod phrase-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY phrase-server/ ./
RUN CGO_ENABLED=0 go build -o /out/phrase-server ./cmd/server
FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=build /out/phrase-server ./phrase-server
ENV APP_ADDR=:19171
EXPOSE 19171
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19171/healthz >/dev/null || exit 1
CMD ["./phrase-server"]

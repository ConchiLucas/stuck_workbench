FROM golang:1.26-alpine AS build
WORKDIR /src/english-server
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY english-server/go.mod english-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY english-server/ ./
RUN CGO_ENABLED=0 go build -o /out/english-server ./cmd/server
FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=build /out/english-server ./english-server
ENV APP_ADDR=:19131
EXPOSE 19131
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19131/healthz >/dev/null || exit 1
CMD ["./english-server"]

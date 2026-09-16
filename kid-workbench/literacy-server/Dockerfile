FROM golang:1.26-alpine AS build

WORKDIR /src/literacy-server
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY literacy-server/go.mod literacy-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY literacy-server/ ./
RUN CGO_ENABLED=0 go build -o /out/literacy-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=build /out/literacy-server ./literacy-server
ENV APP_ADDR=:19151
EXPOSE 19151
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19151/healthz >/dev/null || exit 1
CMD ["./literacy-server"]

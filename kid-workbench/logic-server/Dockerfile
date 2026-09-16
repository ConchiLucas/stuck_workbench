FROM golang:1.26-alpine AS build
WORKDIR /src/logic-server
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY logic-server/go.mod logic-server/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY logic-server/ ./
RUN CGO_ENABLED=0 go build -o /out/logic-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/logic-server ./logic-server
USER app
ENV APP_ADDR=:19191
EXPOSE 19191
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19191/healthz >/dev/null || exit 1
CMD ["./logic-server"]

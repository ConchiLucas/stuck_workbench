# syntax=docker/dockerfile:1
FROM node:22-alpine AS frontend
WORKDIR /build/task-admin/frontend
COPY packages/ /build/packages/
COPY task-admin/frontend/package.json task-admin/frontend/package-lock.json ./
RUN npm ci --install-links
COPY task-admin/frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/task-admin/backend
COPY shared-go/ /src/shared-go/
COPY task-admin/backend/go.mod task-admin/backend/go.sum ./
RUN GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct go mod download
COPY task-admin/backend/ ./
COPY --from=frontend /build/task-admin/backend/internal/http/dist ./internal/http/dist
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=backend-build /out/server ./server
ENV APP_ADDR=:19201
EXPOSE 19201
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19201/healthz >/dev/null || exit 1
CMD ["./server"]

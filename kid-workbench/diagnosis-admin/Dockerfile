# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /build/diagnosis-admin/frontend
COPY packages/pinyin-player/ /build/packages/pinyin-player/
COPY packages/math-player/ /build/packages/math-player/
COPY packages/english-player/ /build/packages/english-player/
COPY packages/phrase-player/ /build/packages/phrase-player/
COPY packages/science-player/ /build/packages/science-player/
COPY packages/poem-player/ /build/packages/poem-player/
COPY packages/logic-player/ /build/packages/logic-player/
COPY packages/chengyu-player/ /build/packages/chengyu-player/
COPY diagnosis-admin/frontend/package.json diagnosis-admin/frontend/package-lock.json* diagnosis-admin/frontend/.npmrc ./
RUN npm ci
COPY diagnosis-admin/frontend/ ./
ARG VITE_PROGRESS_URL=http://localhost:19081
ENV VITE_PROGRESS_URL=$VITE_PROGRESS_URL
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/diagnosis-admin/backend
ENV GOPROXY=https://goproxy.cn,direct
COPY shared-go/go.mod shared-go/go.sum* /src/shared-go/
COPY diagnosis-admin/backend/go.mod diagnosis-admin/backend/go.sum ./
RUN go mod download
COPY shared-go/ /src/shared-go/
COPY diagnosis-admin/backend/ ./
COPY --from=frontend /build/diagnosis-admin/backend/internal/http/dist ./internal/http/dist
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=backend-build /out/server ./server
ENV APP_ADDR=:19211
EXPOSE 19211
HEALTHCHECK --interval=5s --timeout=3s --retries=10 \
  CMD wget -qO- http://127.0.0.1:19211/healthz >/dev/null || exit 1
CMD ["./server"]

# syntax=docker/dockerfile:1
FROM node:22-alpine AS frontend
WORKDIR /build/content-admin/frontend
COPY packages/ /build/packages/
COPY content-admin/frontend/package.json content-admin/frontend/package-lock.json ./
RUN npm ci
COPY content-admin/frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/content-admin/backend
COPY shared-go/ /src/shared-go/
COPY content-admin/backend/go.mod content-admin/backend/go.sum ./
RUN GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct go mod download
COPY content-admin/backend/ ./
COPY --from=frontend /build/content-admin/backend/internal/http/dist ./internal/http/dist
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache wget ca-certificates
WORKDIR /app
COPY --from=backend-build /out/server ./server
COPY content-admin/backend/assets/fonts ./assets/fonts
COPY content-admin/backend/assets/pinyin-human-pack ./assets/pinyin-human-pack
ENV PINYIN_SYLLABLE_PACK_DIR=/app/assets/pinyin-human-pack
ENV GLYPH_FONT_PATH=/app/assets/fonts/NotoSerifSC-Regular.otf
ENV APP_ADDR=:19091
EXPOSE 19091
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:19091/healthz >/dev/null || exit 1
CMD ["./server"]

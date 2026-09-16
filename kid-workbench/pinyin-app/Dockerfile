FROM node:22-alpine AS build

WORKDIR /build/pinyin-app

COPY packages/pinyin-player/ /build/packages/pinyin-player/
COPY pinyin-app/package.json pinyin-app/package-lock.json* pinyin-app/.npmrc ./
RUN npm ci
COPY pinyin-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY pinyin-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/pinyin-app/dist /usr/share/nginx/html
EXPOSE 80

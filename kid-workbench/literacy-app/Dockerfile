FROM node:22-alpine AS build

WORKDIR /build/literacy-app

COPY packages/ /build/packages/
COPY literacy-app/package.json literacy-app/package-lock.json* ./
RUN npm ci

COPY literacy-app/ ./
COPY shared-go/handwriting/testdata/ /build/shared-go/handwriting/testdata/
RUN npm run build

FROM nginx:1.27-alpine

COPY literacy-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/literacy-app/dist /usr/share/nginx/html

EXPOSE 80

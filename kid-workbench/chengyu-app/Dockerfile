FROM node:22-alpine AS build
WORKDIR /build/chengyu-app
COPY packages/ /build/packages/
COPY chengyu-app/package.json chengyu-app/package-lock.json ./
RUN npm ci
COPY chengyu-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY chengyu-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/chengyu-app/dist /usr/share/nginx/html
EXPOSE 80

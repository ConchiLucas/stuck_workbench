FROM node:22-alpine AS build
WORKDIR /build/english-app
COPY packages/ /build/packages/
COPY english-app/package.json english-app/package-lock.json ./
RUN npm ci
COPY english-app/ ./
RUN npm run build
FROM nginx:1.27-alpine
COPY english-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/english-app/dist /usr/share/nginx/html
EXPOSE 80

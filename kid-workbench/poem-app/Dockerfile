FROM node:22-alpine AS build
WORKDIR /build/poem-app
COPY packages/ /build/packages/
COPY poem-app/package.json poem-app/package-lock.json ./
RUN npm ci
COPY poem-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY poem-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/poem-app/dist /usr/share/nginx/html
EXPOSE 80

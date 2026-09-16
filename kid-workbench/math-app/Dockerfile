FROM node:22-alpine AS build

WORKDIR /build/math-app
COPY packages/ /build/packages/
COPY math-app/package.json math-app/package-lock.json ./
RUN npm ci
COPY math-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY math-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/math-app/dist /usr/share/nginx/html
EXPOSE 80

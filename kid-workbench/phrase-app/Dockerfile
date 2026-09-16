FROM node:22-alpine AS build
WORKDIR /build/phrase-app
COPY packages/ /build/packages/
COPY phrase-app/package.json phrase-app/package-lock.json ./
RUN npm ci
COPY phrase-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY phrase-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/phrase-app/dist /usr/share/nginx/html
EXPOSE 80

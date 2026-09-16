FROM node:22-alpine AS build
WORKDIR /build/science-app
COPY packages/ /build/packages/
COPY science-app/package.json science-app/package-lock.json ./
RUN npm ci
COPY science-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY science-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/science-app/dist /usr/share/nginx/html
EXPOSE 80

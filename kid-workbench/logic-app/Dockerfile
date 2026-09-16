FROM node:22-alpine AS build
WORKDIR /build/logic-app
COPY packages/ /build/packages/
COPY logic-app/package.json logic-app/package-lock.json ./
RUN npm ci
COPY logic-app/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY logic-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/logic-app/dist /usr/share/nginx/html
EXPOSE 80

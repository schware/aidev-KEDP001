# syntax=docker/dockerfile:1
# nginx image: kedp-nginx. 화면을 build한 정적 파일과 nginx 설정을 담는다(KEDP001-D023).
# nginx 설정은 deploy/nginx에 있으므로 build context `nginx-conf`로 함께 준다:
#   docker build --build-context nginx-conf=../deploy/nginx -t kedp-nginx:<tag> .
ARG NODE_IMAGE=node:24-alpine
ARG NGINX_IMAGE=nginx:stable-alpine

FROM ${NODE_IMAGE} AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

FROM ${NGINX_IMAGE}
RUN apk add --no-cache tzdata
ENV TZ=Asia/Seoul
COPY --from=nginx-conf default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html

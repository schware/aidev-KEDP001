# syntax=docker/dockerfile:1
# Go 수집·API 서버(cmd/server) image: kedp-go. build는 deploy/build.sh로 한다(KEDP001-D023).
# 실행 image에는 실행 파일과 HTTPS용 CA 인증서만 둔다. 시간대 정보는 실행 파일에 들어 있다(time/tzdata).
ARG GO_IMAGE=golang:1.27-alpine

FROM ${GO_IMAGE} AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/kedp-server ./cmd/server

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/kedp-server /kedp-server
ENV TZ=Asia/Seoul
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/kedp-server"]

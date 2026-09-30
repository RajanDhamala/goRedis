FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /go_redis .

FROM alpine:3.23

RUN addgroup -S -g 10001 appgroup \
    && adduser -S -u 10001 -G appgroup appuser \
    && mkdir /data \
    && chown appuser:appgroup /data

COPY --from=build /go_redis /usr/local/bin/go_redis

WORKDIR /data
ENV HOST=0.0.0.0 PORT=6379
USER 10001:10001
EXPOSE 6379
VOLUME ["/data"]
CMD ["go_redis"]

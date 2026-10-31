FROM golang:1.27.3-alpine3.24 AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dropfarm .

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 dropfarm \
    && adduser -S -D -H -u 10001 -G dropfarm dropfarm \
    && mkdir -p /data/logs \
    && chown -R dropfarm:dropfarm /data

ENV TZ=Europe/Moscow
COPY --from=build --chown=dropfarm:dropfarm /out/dropfarm /app/dropfarm

USER 10001:10001
WORKDIR /data
ENTRYPOINT ["/app/dropfarm"]

FROM golang:1.20.5-alpine3.18 AS builder

WORKDIR /app
COPY . /app

RUN go test ./...
RUN go build -o myq-teslamate-geofence cmd/app/main.go

FROM alpine:3.18

ARG USER_UID=10000
ARG USER_GID=$USER_UID

VOLUME [ "/app/config" ]
WORKDIR /app

RUN apk add --no-cache bash && \
    addgroup --gid $USER_GID nonroot && \
    adduser --uid $USER_UID --ingroup nonroot --system --shell bin/bash nonroot && \
    chown -R nonroot:nonroot /app

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /app/myq-teslamate-geofence /app/config.example.yml /app/

ENV PATH="/app:${PATH}"

USER nonroot

CMD [ "/app/myq-teslamate-geofence", "-c", "/app/config/config.yml" ]

FROM golang:1.20.5-alpine3.18 AS builder

ARG BUILD_VERSION

RUN test -n "${BUILD_VERSION}" || (echo "Build argument BUILD_VERSION is required but not provided" && exit 1)

WORKDIR /app
COPY . ./

RUN go test ./...
RUN go build -ldflags="-X main.version=${BUILD_VERSION}" -o tesla-youq cmd/app/main.go

FROM alpine:3.18

ARG USER_UID=10000
ARG USER_GID=$USER_UID
# store original userid and groupid as env vars to pass to entrypoint for replacement if puid and pgid are specified at runtime
ENV OUID $USER_UID
ENV OGID $USER_GID

VOLUME [ "/app/config" ]
WORKDIR /app

RUN apk add --no-cache bash tzdata su-exec && \
    addgroup --gid $USER_GID nonroot && \
    adduser --uid $USER_UID --ingroup nonroot --system --shell bin/bash nonroot && \
    chown -R nonroot:nonroot /app

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /app/tesla-youq /app/config.example.yml /app/
COPY ./entrypoint.sh /app/

ENV PATH="/app:${PATH}"

ENTRYPOINT [ "/app/entrypoint.sh" ]
CMD [ "/app/tesla-youq", "-c", "/app/config/config.yml" ]

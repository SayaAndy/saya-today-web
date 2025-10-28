FROM golang:1.25.3-alpine3.22 AS build-stage

RUN apk add --no-cache sqlite-dev musl-dev gcc

WORKDIR /builddir
COPY . .
RUN CGO_ENABLED=1 go build -o sayana-web .

FROM alpine:3.22.2 AS runtime-stage

ENV ENVIRONMENT=""
ENV AUTH_SALT=""
ENV B2_KEY_ID=""
ENV B2_APPLICATION_KEY=""

WORKDIR /app
COPY --from=build-stage /builddir/sayana-web /app/sayana-web
COPY --from=build-stage /builddir/migrations /app/migrations
COPY --from=build-stage /builddir/static /app/static
COPY --from=build-stage /builddir/views /app/views
COPY --from=build-stage /builddir/locale/*.yaml /app/locale/
COPY --from=build-stage /builddir/config/config*.yaml /app/config/

RUN apk add --no-cache tzdata sqlite

ENTRYPOINT /app/sayana-web -c /app/config/config.${ENVIRONMENT}.yaml

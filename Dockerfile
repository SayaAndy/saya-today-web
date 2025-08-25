FROM golang:1.24.5-alpine3.22 AS build-stage

WORKDIR /builddir
COPY . .
RUN go build -o sayana-web .

FROM alpine:3.22.1 AS runtime-stage

ENV ENVIRONMENT=""
ENV AUTH_SALT=""
ENV B2_KEY_ID=""
ENV B2_APPLICATION_KEY=""

WORKDIR /app
COPY --from=build-stage /builddir/sayana-web /app/sayana-web
COPY --from=build-stage /builddir/static /app/static
COPY --from=build-stage /builddir/views /app/views
COPY --from=build-stage /builddir/locale/*.yaml /app/locale/
COPY --from=build-stage /builddir/config/config*.yaml /app/config/

RUN apk add --no-cache tzdata

ENTRYPOINT /app/sayana-web -c /app/config/config.${ENVIRONMENT}.yaml

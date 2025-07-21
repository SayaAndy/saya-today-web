FROM golang:1.24.5-alpine3.22 AS build-stage

WORKDIR /builddir
COPY . .
RUN go build -o sayana-demo .

FROM alpine:3.22.1 AS runtime-stage

WORKDIR /app
COPY --from=build-stage /builddir/sayana-demo /app/sayana-demo
COPY --from=build-stage /builddir/public /app/public

ENTRYPOINT [ "/app/sayana-demo" ]

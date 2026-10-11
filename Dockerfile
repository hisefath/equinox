# Demo deploy (Railway): serves the page and API from the recorded 2026-10-11 venue responses; no live venue calls.
FROM golang:1.27.2-alpine3.24 AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /equinox ./cmd/equinox

FROM alpine:3.24
WORKDIR /app
COPY --from=build /equinox ./equinox
COPY refdata ./refdata
COPY testdata/live-20261011 ./testdata/live-20261011
RUN adduser -D app && mkdir data && chown app data
USER app
CMD ["sh", "-c", "exec ./equinox serve -addr :${PORT:-8080}"]

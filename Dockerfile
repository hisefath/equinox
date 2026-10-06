# Build a static binary, then ship it on distroless: no shell, no package manager, runs as non-root.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /equinox ./cmd/equinox && mkdir -p /app/data

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /equinox /usr/local/bin/equinox
COPY --from=build --chown=65532:65532 /app/data /app/data
# The reviewed mapping table and a recorded live snapshot, so `-replay testdata/snapshot` works offline.
COPY reviews /app/reviews
COPY testdata/snapshot /app/testdata/snapshot
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/equinox"]
CMD ["serve"]

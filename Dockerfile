# Build a static binary, then ship it on distroless (no shell, no package manager, non-root).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /equinox ./cmd/equinox

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /equinox /equinox
COPY testdata/snapshot /snapshot
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/equinox"]
CMD ["serve"]

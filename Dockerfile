FROM --platform=$BUILDPLATFORM node:24.22.3-alpine3.24 AS web
WORKDIR /src/frontend
RUN npm install -g pnpm@12.15.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24 AS build
ARG TARGETOS
ARG TARGETARCH
# Versión que muestra `pmt version`; por ejemplo --build-arg PMT_VERSION=$(git describe --tags --always).
ARG PMT_VERSION=dev
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/pmt-server ./cmd/pmt-server
# El CLI pmt para los cuatro sistemas, con SHA256SUMS; pmt-server lo sirve desde cli.dist.
RUN scripts/build-cli.sh /out/cli "$PMT_VERSION"

FROM alpine:3.24.4
WORKDIR /app
COPY --from=build --chown=65532:65532 /out/pmt-server /app/pmt-server
COPY --chown=65532:65532 backend/config/config.prod.yaml /app/config/config.prod.yaml
COPY --from=web --chown=65532:65532 /src/frontend/dist /app/web
COPY --from=build --chown=65532:65532 /out/cli /app/cli
ENV ENV=prod FRONTEND_DIST=/app/web
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD wget -qO- http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/app/pmt-server"]

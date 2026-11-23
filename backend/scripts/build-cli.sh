#!/bin/sh
# Compila el CLI pmt para Linux y macOS, amd64 y arm64, con la versión incrustada,
# y escribe SHA256SUMS. pmt-server los sirve desde cli.dist.
#
#   scripts/build-cli.sh [carpeta] [versión]
#
# Por omisión: ../dist/cli (la que usa config.dev.yaml) y la versión de
# `git describe`, o PMT_VERSION si está definida. Se corre desde backend/.
set -eu

cd "$(dirname "$0")/.."
salida="${1:-../dist/cli}"
version="${2:-${PMT_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}}"

mkdir -p "$salida"
for objetivo in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
	so="${objetivo%/*}"
	arq="${objetivo#*/}"
	echo "pmt-$so-$arq ($version)"
	CGO_ENABLED=0 GOOS="$so" GOARCH="$arq" go build -trimpath \
		-ldflags "-s -w -X pmt/internal/cli.Version=$version" \
		-o "$salida/pmt-$so-$arq.nuevo" ./cmd/pmt
	mv -f "$salida/pmt-$so-$arq.nuevo" "$salida/pmt-$so-$arq"
done

cd "$salida"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum pmt-linux-amd64 pmt-linux-arm64 pmt-darwin-amd64 pmt-darwin-arm64 > SHA256SUMS.nuevo
else
	shasum -a 256 pmt-linux-amd64 pmt-linux-arm64 pmt-darwin-amd64 pmt-darwin-arm64 > SHA256SUMS.nuevo
fi
mv -f SHA256SUMS.nuevo SHA256SUMS
echo "Listo en $(pwd)"

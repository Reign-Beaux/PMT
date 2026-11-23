#!/bin/sh
# Instala el CLI pmt en ~/.local/bin/pmt, sin sudo. Lo sirve pmt-server en
# /install.sh con su URL pública ya puesta:
#   curl -fsSL __PMT_SERVER__/install.sh | sh
set -eu

SERVIDOR="__PMT_SERVER__"
DESTINO="${HOME:?falta HOME}/.local/bin"

falla() {
	echo "pmt: $*" >&2
	exit 1
}

case "$(uname -s)" in
	Linux) so=linux ;;
	Darwin) so=darwin ;;
	*) falla "sistema no soportado: $(uname -s). Hay binarios para Linux y macOS." ;;
esac
case "$(uname -m)" in
	x86_64 | amd64) arq=amd64 ;;
	aarch64 | arm64) arq=arm64 ;;
	*) falla "arquitectura no soportada: $(uname -m). Hay binarios para amd64 y arm64." ;;
esac
binario="pmt-$so-$arq"

if command -v curl >/dev/null 2>&1; then
	bajar() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	bajar() { wget -q -O "$2" "$1"; }
else
	falla "necesito curl o wget para descargar."
fi

if command -v sha256sum >/dev/null 2>&1; then
	suma() { sha256sum "$1" | cut -d ' ' -f 1; }
elif command -v shasum >/dev/null 2>&1; then
	suma() { shasum -a 256 "$1" | cut -d ' ' -f 1; }
else
	falla "necesito sha256sum o shasum para verificar la descarga."
fi

temporal="$(mktemp -d 2>/dev/null || mktemp -d -t pmt)"
trap 'rm -rf "$temporal"' EXIT INT TERM

echo "Descargando $binario de $SERVIDOR…"
bajar "$SERVIDOR/descargas/$binario" "$temporal/pmt" || falla "no se pudo descargar $SERVIDOR/descargas/$binario"
bajar "$SERVIDOR/descargas/SHA256SUMS" "$temporal/SHA256SUMS" || falla "no se pudo descargar $SERVIDOR/descargas/SHA256SUMS"

esperada="$(awk -v f="$binario" '$2 == f || $2 == "*" f { print $1 }' "$temporal/SHA256SUMS")"
[ -n "$esperada" ] || falla "SHA256SUMS no tiene la suma de $binario"
obtenida="$(suma "$temporal/pmt")"
[ "$esperada" = "$obtenida" ] || falla "la suma SHA-256 de $binario no coincide; no se instaló nada"

mkdir -p "$DESTINO"
chmod 755 "$temporal/pmt"
mv -f "$temporal/pmt" "$DESTINO/pmt"
echo "pmt instalado en $DESTINO/pmt ($("$DESTINO/pmt" version))"

case ":${PATH:-}:" in
	*":$DESTINO:"*) ;;
	*)
		echo
		echo "Aviso: $DESTINO no está en tu PATH. Agrégalo a tu ~/.profile, ~/.bashrc o ~/.zshrc:"
		echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
		;;
esac
echo
echo "Siguiente paso: pmt login"

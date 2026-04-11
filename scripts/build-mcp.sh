#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

go build -o "$ROOT/bin/pmt-mcp" "$ROOT/cmd/mcp"

echo "built: bin/pmt-mcp"

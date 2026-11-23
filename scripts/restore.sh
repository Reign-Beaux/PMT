#!/usr/bin/env bash
set -euo pipefail
usage() { echo 'uso: restore.sh <archivo> <base> [--yes]' >&2; exit 2; }
(($# == 2 || $# == 3)) || usage
[[ $# == 2 || $3 == --yes ]] || usage
file=$1
base=$2
container=${BACKUP_CONTAINER:-}
db_user=${BACKUP_DB_USER:-pmt}
[[ -n $container ]] || { echo 'restore: falta BACKUP_CONTAINER' >&2; exit 1; }
[[ -f $file && -s $file && -r $file ]] || { echo "restore: archivo inexistente, vacío o ilegible: $file" >&2; exit 1; }
if (($# == 2)); then
  printf 'Escribe el nombre de la base %s para confirmar: ' "$base" >&2
  IFS= read -r answer || answer=''
  [[ $answer == "$base" ]] || { echo 'restore: cancelado' >&2; exit 1; }
fi
if ! docker exec -i "$container" pg_restore -U "$db_user" -d "$base" --no-owner --no-privileges --clean --if-exists --single-transaction < "$file"; then
  echo 'restore: falló la restauración' >&2
  exit 1
fi
printf 'Restauración completada en %s\n' "$base"

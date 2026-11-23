#!/usr/bin/env bash
set -euo pipefail

fail() { echo "backup: $*" >&2; exit 1; }
container=${BACKUP_CONTAINER:-}
db_user=${BACKUP_DB_USER:-pmt}
db=${BACKUP_DB:-pmt}
dir=${BACKUP_DIR:-./backups}
keep=${BACKUP_KEEP:-14}
[[ -n $container ]] || fail 'falta BACKUP_CONTAINER'
[[ $keep =~ ^[0-9]+$ ]] && [[ ${keep//0/} != '' ]] || fail 'BACKUP_KEEP debe ser un entero mayor que cero'
keep=${keep#"${keep%%[!0]*}"}
# Evitar nombres de archivo ambiguos y opciones de comandos.
[[ $db != */* && $db != . && $db != .. && $db != -* ]] || fail 'BACKUP_DB no es válido para un nombre de archivo'
mkdir -p -- "$dir" || fail "no se pudo crear $dir"
file="$dir/$db-$(date +%Y%m%d-%H%M%S).dump"
tmp="$file.tmp"
[[ ! -e $file && ! -e $tmp ]] || fail "ya existe $file o su archivo temporal"
trap 'rm -f -- "$tmp"' EXIT
if ! docker exec "$container" pg_dump -Fc -U "$db_user" -d "$db" > "$tmp"; then
  fail 'falló pg_dump o el contenedor no existe'
fi
mv -n -- "$tmp" "$file" || fail 'no se pudo guardar el respaldo'
[[ ! -e $tmp ]] || fail "ya existe $file"
trap - EXIT

shopt -s nullglob
files=("$dir/$db-"????????-??????.dump)
matched=()
for candidate in "${files[@]}"; do
  suffix=${candidate#"$dir/$db-"}
  [[ $suffix =~ ^[0-9]{8}-[0-9]{6}\.dump$ ]] && matched+=("$candidate")
done
# El patrón fijo y LC_ALL=C ordenan cronológicamente por nombre.
if ((${#matched[@]} > 0)); then
  mapfile -d '' -t matched < <(printf '%s\0' "${matched[@]}" | LC_ALL=C sort -z)
  # Limitar la conversión numérica evita desbordamientos con valores enormes.
  if ((${#keep} < 10)) && ((${#matched[@]} > 10#$keep)); then
    remove=$((${#matched[@]} - 10#$keep))
    rm -f -- "${matched[@]:0:remove}"
  fi
fi
printf '%s\n' "$file"

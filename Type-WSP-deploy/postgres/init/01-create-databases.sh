#!/bin/sh
set -eu

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_DB_SYSTEM:?POSTGRES_DB_SYSTEM is required}"
: "${POSTGRES_DB_USER:?POSTGRES_DB_USER is required}"

if [ "$POSTGRES_DB" != "$POSTGRES_DB_SYSTEM" ]; then
  echo "POSTGRES_DB must match POSTGRES_DB_SYSTEM" >&2
  return 1 2>/dev/null || exit 1
fi

psql \
  --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB_SYSTEM" \
  --set=user_database="$POSTGRES_DB_USER" \
  --set=database_owner="$POSTGRES_USER" <<'EOSQL'
SELECT format(
  'CREATE DATABASE %I OWNER %I',
  :'user_database',
  :'database_owner'
)
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_database
  WHERE datname = :'user_database'
)
\gexec
EOSQL

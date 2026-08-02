#!/bin/sh
set -eu

certificate_path="${TLS_CERTIFICATE_PATH:?TLS_CERTIFICATE_PATH is required}"
private_key_path="${TLS_PRIVATE_KEY_PATH:?TLS_PRIVATE_KEY_PATH is required}"
server_name="${SERVER_NAME:?SERVER_NAME is required}"

case "$server_name" in
  localhost|*.localhost|127.0.0.1|::1)
    echo "Refusing to start TLS with a localhost server name." >&2
    exit 1
    ;;
esac

for path in "$certificate_path" "$private_key_path"; do
  if [ ! -r "$path" ]; then
    echo "Required TLS file is unreadable: $path" >&2
    exit 1
  fi
done

if ! openssl x509 -in "$certificate_path" -noout >/dev/null 2>&1; then
  echo "TLS certificate is not a valid X.509 PEM file." >&2
  exit 1
fi

if ! openssl pkey -in "$private_key_path" -noout >/dev/null 2>&1; then
  echo "TLS private key is not a valid PEM key." >&2
  exit 1
fi

subject="$(openssl x509 -in "$certificate_path" -noout -subject -nameopt RFC2253 | sed 's/^subject=//')"
issuer="$(openssl x509 -in "$certificate_path" -noout -issuer -nameopt RFC2253 | sed 's/^issuer=//')"
if [ "$subject" = "$issuer" ]; then
  echo "Refusing to start with a self-signed TLS certificate." >&2
  exit 1
fi

if ! openssl x509 -in "$certificate_path" -noout -checkhost "$server_name" >/dev/null 2>&1; then
  echo "TLS certificate does not match SERVER_NAME=$server_name." >&2
  exit 1
fi

if ! openssl x509 -in "$certificate_path" -noout -checkend 2592000 >/dev/null 2>&1; then
  echo "TLS certificate expires within 30 days or is already expired." >&2
  exit 1
fi

certificate_public_key="$(openssl x509 -in "$certificate_path" -pubkey -noout | openssl pkey -pubin -outform DER | openssl dgst -sha256)"
private_key_public_key="$(openssl pkey -in "$private_key_path" -pubout -outform DER | openssl dgst -sha256)"
if [ "$certificate_public_key" != "$private_key_public_key" ]; then
  echo "TLS certificate and private key do not match." >&2
  exit 1
fi

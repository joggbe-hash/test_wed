#!/usr/bin/env bash
# ======================================================
# 產生本機開發用自簽 SSL 憑證 (localhost + 127.0.0.1)
# 用法: bash generate-dev-cert.sh
# 正式上線請換成 Let's Encrypt / 正式 CA 簽發的憑證
# ======================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [[ -f fullchain.pem && -f privkey.pem ]]; then
    echo "[!] 憑證已存在,如需重新產生請先手動刪除 fullchain.pem 與 privkey.pem"
    exit 0
fi

cat > openssl-san.cnf <<'EOF'
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
req_extensions     = v3_req
distinguished_name = dn

[dn]
C  = TW
ST = Taiwan
L  = Taipei
O  = DevLocal
CN = localhost

[v3_req]
subjectAltName = @alt_names
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1  = 127.0.0.1
IP.2  = ::1
EOF

openssl req -x509 -nodes -days 365 \
    -newkey rsa:2048 \
    -keyout privkey.pem \
    -out fullchain.pem \
    -config openssl-san.cnf \
    -extensions v3_req

rm -f openssl-san.cnf
chmod 644 fullchain.pem
chmod 600 privkey.pem

echo "[+] 已產生:"
echo "    $SCRIPT_DIR/fullchain.pem"
echo "    $SCRIPT_DIR/privkey.pem"
echo "[i] 瀏覽器會警告「不受信任」,點「進階 -> 繼續前往」即可(僅限本機測試)"

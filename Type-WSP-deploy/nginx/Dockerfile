FROM node:24-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd AS frontend-builder

WORKDIR /frontend

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html vite.config.ts tsconfig.json tsconfig.node.json ./
COPY public ./public
COPY src ./src

ARG VITE_SITE_URL=
ARG VITE_BASE_PATH=/
ARG VITE_API_BASE_URL=
ENV VITE_SITE_URL=$VITE_SITE_URL
ENV VITE_BASE_PATH=$VITE_BASE_PATH
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run typecheck \
    && npm exec vite build -- --mode backend --outDir=/frontend-dist --emptyOutDir

FROM owasp/modsecurity-crs:nginx@sha256:2051ff18b836c1d9bbc5c7754451c1687ea27352e497b89d0c9fc7e657861e07

USER root

RUN if command -v apk >/dev/null 2>&1; then \
        apk upgrade --no-cache && apk add --no-cache openssl; \
    else \
        apt-get update \
        && apt-get upgrade -y \
        && apt-get install -y --no-install-recommends openssl \
        && rm -rf /var/lib/apt/lists/*; \
    fi

COPY Type-WSP-deploy/nginx/scripts/10-validate-tls.sh /docker-entrypoint.d/10-validate-tls.sh
RUN chmod 755 /docker-entrypoint.d/10-validate-tls.sh

COPY --from=frontend-builder /frontend-dist /usr/share/nginx/html

USER nginx

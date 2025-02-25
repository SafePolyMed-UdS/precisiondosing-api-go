FROM golang:1.23.1-alpine3.20 AS builder

WORKDIR /app

COPY api/ ./
COPY .git/ ./
RUN apk update && apk add --no-cache git && \
    go mod tidy && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init && \
    swag fmt && go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.versionTag=$(git rev-parse --short HEAD)" -trimpath -o api .

FROM rocker/r-ver:4.3.3

LABEL org.opencontainers.image.source=https://github.com/SafePolyMed-UdS/precisiondosing-api-go
LABEL org.opencontainers.image.description="Safepolymed Precisiondosing API"
LABEL org.opencontainers.image.licenses=MIT

ARG DOCKERBASE=rbase-dockerfiles
ARG INSTALL_OSP=true
ARG PROD_BUILD=true

WORKDIR /
COPY ${DOCKERBASE}/scripts/install_base_pkg.sh \
    ${DOCKERBASE}/scripts/install_osp_pkg.sh \
    ${DOCKERBASE}/scripts/install_user_r_pkg.sh \
    /setup/

# Install base packages for ubuntu
RUN chmod +x /setup/*.sh && \
    setup/install_base_pkg.sh

# Optional install of OSPSuite packages and dependencies
RUN setup/install_osp_pkg.sh

# Optional install of user R packages
#COPY ${R_PKG_FILE} rbase-dockerfiles/scripts/packages.R
#RUN rbase-dockerfiles/scripts/install_user_r_pkg.sh

RUN mkdir -p /app /app/schemas /logs && \
    chown -R appuser:appuser /app /logs \
    && chmod -R 755 /app \
    && chmod -R 755 /logs

COPY --from=builder /app/api /app/api
COPY api/cfg/prod_config.yml /app/config.yml
COPY api/schemas/* /app/schemas/

USER appuser
WORKDIR /app
ENTRYPOINT ["/app/api", "--config", "/app/config.yml"]

EXPOSE 3332

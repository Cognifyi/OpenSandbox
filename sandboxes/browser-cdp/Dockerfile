# syntax=docker/dockerfile:1
# Copyright 2025 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Multi-stage build: Build execd from source, then use it in the final image
# Build command: docker build -t opensandbox/browser-cdp:latest -f sandboxes/browser-cdp/Dockerfile .

# Stage 1: Build execd from source
FROM golang:1.25.9 AS execd-builder

WORKDIR /build

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Prepare local modules to satisfy replace directives
COPY components/internal/go.mod components/internal/go.sum ./components/internal/
COPY components/execd/go.mod components/execd/go.sum ./components/execd/

# Download deps with only mod files for better caching
RUN cd components/internal && go mod download
RUN cd components/execd && go mod download

# Copy sources
COPY components/internal ./components/internal
COPY components/execd ./components/execd

WORKDIR /build/components/execd

RUN CGO_ENABLED=0 go build -mod=mod \
    -ldflags "-X 'github.com/alibaba/opensandbox/internal/version.Version=${VERSION}' \
              -X 'github.com/alibaba/opensandbox/internal/version.BuildTime=${BUILD_TIME}' \
              -X 'github.com/alibaba/opensandbox/internal/version.GitCommit=${GIT_COMMIT}'" \
    -o /build/execd ./main.go

# Stage 2: Final image with code-interpreter base
FROM opensandbox/code-interpreter-base:latest

ARG TARGETARCH

# Copy execd from the builder stage
COPY --from=execd-builder /build/execd /usr/local/bin/execd
RUN chmod +x /usr/local/bin/execd

# Install Xvfb and browser dependencies (following browserless approach)
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    dumb-init \
    fonts-liberation \
    libasound2t64 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libwayland-client0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxkbcommon0 \
    libxrandr2 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

# Set Playwright browsers path before installation (following browserless approach)
ENV PLAYWRIGHT_BROWSERS_PATH=/usr/local/bin/playwright-browsers \
    PUPPETEER_SKIP_DOWNLOAD=true

# Install Playwright and Chromium
RUN source /opt/opensandbox/code-interpreter-env.sh node 22 && \
    npm install -g playwright && \
    npx playwright install chromium --with-deps

# Copy scripts
COPY scripts/browser-launch.sh /opt/opensandbox/browser-launch.sh
COPY scripts/browser-entrypoint.sh /opt/opensandbox/browser-entrypoint.sh
RUN chmod +x /opt/opensandbox/browser-launch.sh /opt/opensandbox/browser-entrypoint.sh

# Environment variables
ENV EXECD_PORT=44772 \
    BROWSER_LAUNCH_ARGS="--no-sandbox --disable-dev-shm-usage" \
    BROWSER_HEADLESS=true \
    BROWSER_CDP_PORT=0 \
    ENABLE_OVERLAYFS_SNAPSHOTS=false \
    BROWSER_OVERLAYFS_SNAPSHOT_BACKEND="" \
    PLAYWRIGHT_BROWSERS_PATH=/usr/local/bin/playwright-browsers \
    NODE_VERSION=22

WORKDIR /workspace
ENTRYPOINT ["/opt/opensandbox/browser-entrypoint.sh"]

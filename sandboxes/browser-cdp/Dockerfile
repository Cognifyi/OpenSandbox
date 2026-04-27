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


ARG EXECD_VERSION=latest

FROM opensandbox/execd:${EXECD_VERSION} AS execd-stage

FROM opensandbox/browser-cdp-base:latest

ARG TARGETARCH

COPY --from=execd-stage /execd /usr/local/bin/execd
RUN chmod +x /usr/local/bin/execd

# Copy scripts
COPY scripts/browser-launch.sh /opt/opensandbox/browser-launch.sh
COPY scripts/browser-entrypoint.sh /opt/opensandbox/browser-entrypoint.sh
RUN chmod +x /opt/opensandbox/browser-launch.sh /opt/opensandbox/browser-entrypoint.sh

# Environment variables
ENV EXECD_PORT=8080 \
    EXECD_ACCESS_TOKEN= \
    BROWSER_LAUNCH_ARGS="--no-sandbox --disable-dev-shm-usage --disable-gpu --disable-software-rasterizer --disable-background-networking --disable-default-apps --disable-extensions --disable-sync --disable-translate --no-first-run --safebrowsing-disable-auto-update" \
    BROWSER_CDP_PORT=0 \
    ENABLE_OVERLAYFS_SNAPSHOTS=false

WORKDIR /workspace
ENTRYPOINT ["/opt/opensandbox/browser-entrypoint.sh"]

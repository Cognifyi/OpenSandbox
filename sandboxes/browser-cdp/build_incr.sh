# 1. 构建新镜像（先给个中间标签，避免直接覆盖）
docker build -t opensandbox/browser-cdp:new .

# 2. 把新镜像打上 latest 标签，覆盖原来的 latest
docker tag opensandbox/browser-cdp:new opensandbox/browser-cdp:latest
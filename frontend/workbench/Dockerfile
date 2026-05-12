FROM docker.1ms.run/node:20-alpine AS builder
RUN npm install -g bun --registry=https://registry.npmmirror.com
WORKDIR /app
COPY package.json bun.lock* ./
RUN echo '[install]\nregistry = "https://registry.npmmirror.com"' > .bunfig.toml
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile || bun install
COPY . .
RUN bun run build

FROM docker.1ms.run/nginx:alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html/app
RUN chmod -R 755 /usr/share/nginx/html/ && \
    chown -R nginx:nginx /usr/share/nginx/html/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

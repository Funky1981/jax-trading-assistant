# Build stage
FROM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS builder

WORKDIR /app

# Copy package files first for layer caching
COPY package.json package-lock.json ./
RUN npm ci --prefer-offline

# Copy source and build
COPY . .
ARG VITE_JAX_API_URL=http://localhost:8081
ENV VITE_JAX_API_URL=${VITE_JAX_API_URL}
RUN npm run build

# Runtime stage — nginx to serve static assets
FROM nginx:1.27-alpine@sha256:65645c7bb6a0661892a8b03b89d0743208a18dd2f3f17a54ef4b76fb8e2f2a10 AS runtime

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# nginx config: serve SPA, proxy /api and /health to jax-trader
# The API_URL is resolved at runtime via envsubst
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost/health-frontend || exit 1

CMD ["nginx", "-g", "daemon off;"]

# Stage 1: Build the React application
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Set build-time argument for the API URL
ARG REACT_APP_API_URL

# --- Caching Optimization ---
# Copy only package files first
COPY package.json package-lock.json* ./
# Or if using yarn:
# COPY package.json yarn.lock ./

# Install dependencies based *only* on lock file changes
# This layer is cached if package*.json haven't changed
RUN echo "Installing dependencies..." && npm ci --omit=dev
# Using --omit=dev can slightly speed up install and reduce image size
# Or if using yarn:
# RUN yarn install --frozen-lockfile --production

# --- Copy Source Code ---
# Copy source code *after* installing dependencies
# Now, changes to source code won't cause re-installation of node_modules
COPY public ./public
COPY src ./src
# Copy any other necessary config files here if needed (e.g., tsconfig.json, craco.config.js)
# COPY tsconfig.json ./
# COPY craco.config.js ./

# --- Build Step ---
# This layer is cached if source code (public, src) hasn't changed
# Ensure REACT_APP_API_URL is available during the build
RUN echo "Building React App... Using API URL: ${REACT_APP_API_URL}" && \
    REACT_APP_API_URL=${REACT_APP_API_URL} npm run build
# Or if using yarn:
# RUN REACT_APP_API_URL=${REACT_APP_API_URL} yarn build

# Stage 2: Serve static files with Caddy
FROM caddy:2-alpine

WORKDIR /usr/share/caddy

# Remove default Caddy config
RUN rm /etc/caddy/Caddyfile

# Copy custom Caddyfile (Only needs Caddyfile, not other source)
COPY Caddyfile /etc/caddy/Caddyfile

# Copy *only* the built assets from the builder stage
COPY --from=builder /app/build .

# Expose the port Caddy listens on (matching fly.toml internal_port)
EXPOSE 8080

# Caddy will automatically start and serve using the Caddyfile
# No specific CMD needed unless customizing Caddy's startup
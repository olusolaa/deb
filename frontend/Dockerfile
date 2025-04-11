# Stage 1: Build the React application
FROM node:20-alpine AS builder

WORKDIR /app

# Set build-time argument for the API URL
ARG REACT_APP_API_URL

# Copy package.json and package-lock.json (or yarn.lock)
COPY package*.json ./
# Or if using yarn:
# COPY package.json yarn.lock ./

# Install dependencies
RUN npm ci
# Or if using yarn:
# RUN yarn install --frozen-lockfile

# Copy the rest of the application code
COPY . .

# Build the React app, injecting the API URL
# Ensure REACT_APP_API_URL is available during the build
RUN echo "Using API URL: ${REACT_APP_API_URL}" && \
    REACT_APP_API_URL=${REACT_APP_API_URL} npm run build
# Or if using yarn:
# RUN REACT_APP_API_URL=${REACT_APP_API_URL} yarn build


# Stage 2: Serve static files with Caddy
FROM caddy:2-alpine

WORKDIR /usr/share/caddy

# Remove default Caddy config
RUN rm /etc/caddy/Caddyfile

# Copy custom Caddyfile
COPY Caddyfile /etc/caddy/Caddyfile

# Copy built React app files from builder stage
COPY --from=builder /app/build .

# Caddy's default port is 80, but fly.toml maps external 80/443 to internal 8080
# So Caddyfile should listen on :8080
EXPOSE 8080
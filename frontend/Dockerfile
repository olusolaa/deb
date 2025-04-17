# Production Dockerfile for Bible App Frontend
FROM node:20-alpine AS build

# Set working directory
WORKDIR /app

# Define build arguments (received from Render)
ARG REACT_APP_GOOGLE_CLIENT_ID
ARG REACT_APP_API_URL

# Make build arguments available as environment variables during build
ENV REACT_APP_GOOGLE_CLIENT_ID=$REACT_APP_GOOGLE_CLIENT_ID
ENV REACT_APP_API_URL=$REACT_APP_API_URL

# Copy package files
COPY package.json package-lock.json* ./

# Install only production dependencies to reduce image size
RUN npm ci --production=false

# Copy source code
COPY public ./public
COPY src ./src

# Build the application
RUN npm run build

# Second stage: serve the built app with minimal footprint
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Install serve package globally with no unnecessary files
RUN npm install -g serve --no-cache

# Copy built assets from the build stage
COPY --from=build /app/build ./build

# Expose port
EXPOSE 10000

# Set environment variables
ENV NODE_ENV=production

# Set non-root user for security
USER node

# Start server - note that serve binds to 0.0.0.0 by default
CMD ["serve", "-s", "build", "-l", "10000"]
# Development Dockerfile for Bible App Frontend
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Set environment variables
# These will be overridden by docker-compose environment settings
ENV REACT_APP_API_URL=http://localhost:8084
ENV REACT_APP_GOOGLE_CLIENT_ID="735291204555-c4t500b5m8eh9ma98i9p1pq3a5o9us6b.apps.googleusercontent.com"

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source code
COPY public ./public
COPY src ./src

# Expose port for development server
EXPOSE 3000

# Start development server
CMD ["npm", "start"]
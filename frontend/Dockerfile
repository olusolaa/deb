# Development Dockerfile for Bible App Frontend
FROM node:20-alpine

# Set working directory
WORKDIR /app

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
FROM node:14.12.0-stretch

WORKDIR /app
COPY . .
RUN npm install

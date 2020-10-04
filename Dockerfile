FROM node:14.12.0-stretch

ENV NODE_ENV production
WORKDIR /app
COPY . .
RUN npm install

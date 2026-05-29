FROM node:22-alpine AS base

WORKDIR /app

COPY frontend/package.json ./
RUN npm install

COPY frontend/ ./

FROM base AS dev
EXPOSE 5173
CMD ["npm", "run", "dev"]

FROM base AS test
CMD ["npm", "run", "test:run"]

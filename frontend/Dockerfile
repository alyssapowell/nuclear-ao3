# Builds the website into plain files, then puts them in a Caddy image that
# serves them. There's no Node.js in the running site; Node is only used here,
# to build.
FROM node:22.23.3-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

FROM caddy:2.11.4-alpine
COPY --from=build /app/dist /srv

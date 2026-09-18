# Build context is the repository root (the client depends on ../sdk).
# SDK build
FROM node:18 AS sdk-build

WORKDIR /sdk

COPY sdk/package.json sdk/package-lock.json ./

RUN npm ci

COPY sdk/ ./

RUN npm run build

# Client build
FROM node:18 AS client-build

# TURN runtime config: baked into the served turn-config.json from build args
# (overridable at deploy time with a file mount, e.g. Helm ConfigMap).
ARG TURN_PUBLIC_URL
ARG TURN_USER
ARG TURN_PASSWORD
ARG TURN_FORCE_RELAY=false
ENV TURN_PUBLIC_URL=$TURN_PUBLIC_URL \
    TURN_USER=$TURN_USER \
    TURN_PASSWORD=$TURN_PASSWORD \
    TURN_FORCE_RELAY=$TURN_FORCE_RELAY

WORKDIR /app

COPY --from=sdk-build /sdk /sdk

COPY client/package.json client/package-lock.json ./

RUN npm ci

COPY client/ ./

RUN npm run build

RUN node -e "require('fs').writeFileSync('./build/turn-config.json', JSON.stringify({url: process.env.TURN_PUBLIC_URL, username: process.env.TURN_USER, credential: process.env.TURN_PASSWORD, forceRelay: process.env.TURN_FORCE_RELAY === 'true'}))"

# Serving container
FROM nginx:1.27-alpine
COPY --from=client-build /app/build /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

FROM node:20-slim AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /app
COPY ./package.json .
COPY ./pnpm-lock.yaml .
RUN pnpm install --frozen-lockfile
COPY . .
CMD [ "pnpm", "build:w" ]
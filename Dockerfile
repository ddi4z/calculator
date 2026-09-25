FROM golang:1.20-alpine AS backend-builder

WORKDIR /src/backend
COPY backend/go.mod ./
COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/calculator ./cmd/server

FROM node:22-alpine AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM alpine:3.20

RUN apk add --no-cache nginx

COPY --from=backend-builder /out/calculator /usr/local/bin/calculator
COPY --from=frontend-builder /src/frontend/dist /usr/share/nginx/html
COPY docker/nginx.conf /etc/nginx/http.d/default.conf
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

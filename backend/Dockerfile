# Builds the single Nuclear AO3 backend binary.
FROM golang:1.23.4-alpine3.21 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /nuclear ./cmd/nuclear

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1001 app
COPY --from=build /nuclear /usr/local/bin/nuclear
USER app
EXPOSE 8080 9091
CMD ["nuclear"]

# Build stage: compile a static Go binary.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/ingestion ./services/ingestion

# Run stage: no shell, no package manager, non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ingestion /ingestion
USER nonroot
ENTRYPOINT ["/ingestion"]

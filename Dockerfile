# Build the operator binary in a full Go toolchain image, then copy it into a
# minimal distroless image for the final runtime container.

# ---- build stage ------------------------------------------------------------
FROM golang:1.24.1 AS builder

WORKDIR /workspace

# Cache module downloads before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY api/       api/
COPY internal/  internal/
COPY main.go    main.go

# Build a statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags="-s -w" -o manager .

# ---- runtime stage ----------------------------------------------------------
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]

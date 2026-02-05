FROM --platform=$BUILDPLATFORM golang:1.22.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /mongo-doctor

# Final stage – copy static binary
FROM ubuntu
COPY --from=builder /mongo-doctor /mongo-doctor
CMD ["/mongo-doctor"]
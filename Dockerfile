# syntax=docker/dockerfile:1
# ---------------------------------------------------------------------------
# A3K — Assessment · Audit · Analyzer for Kubernetes
#
# Minimal distroless image. The binary is pre-built by GoReleaser.
# Includes CA certificates (required for HTTPS to Kubernetes API).
# Runs as non-root (uid 65532 = nonroot).
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

COPY a3k /usr/local/bin/a3k

# Kubernetes config is mounted at runtime:
#   docker run --rm -v ~/.kube:/home/nonroot/.kube:ro ghcr.io/gustavoesser/a3k health
USER nonroot

ENTRYPOINT ["a3k"]
CMD ["--help"]

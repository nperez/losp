# grammar/Dockerfile — Multi-stage build for the losp syntax checker.
#
# Stage 1: Generate Go parser source from Losp.g4 using ANTLR4.
# Stage 2: Compile the Go syntax checker with the generated parser.
# Stage 3: Minimal runtime image with just the losp-check binary.

# ---------------------------------------------------------------------------
# Stage 1: Generate
# ---------------------------------------------------------------------------
FROM eclipse-temurin:17-jre-jammy AS generate

WORKDIR /work

# Download ANTLR4 tool.
ADD https://www.antlr.org/download/antlr-4.13.2-complete.jar /work/antlr4.jar

COPY Losp.g4 .

# Generate Go parser into parser/ subdirectory.
RUN java -jar antlr4.jar \
    -Dlanguage=Go \
    -package parser \
    -o parser \
    Losp.g4

# ---------------------------------------------------------------------------
# Stage 2: Build
# ---------------------------------------------------------------------------
FROM golang:1.24-bookworm AS build

WORKDIR /work

# Copy module file first for better caching.
COPY go.mod .
COPY main.go .
COPY --from=generate /work/parser ./parser

# Resolve dependencies and build.
RUN go mod tidy && go build -o losp-check .

# ---------------------------------------------------------------------------
# Stage 3: Runtime
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim

COPY --from=build /work/losp-check /usr/local/bin/losp-check

ENTRYPOINT ["losp-check"]

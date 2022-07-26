# syntax=docker/dockerfile:1
FROM golang:1.16-alpine3.15@sha256:9743f230f26d1e300545f0330fd4a514f554c535d967563ee77bf634906502b6 as builder

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

# Statically compile our app for use in a distroless container
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -v -o action .

# A distroless container image with some basics like SSL certificates
# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static@sha256:57f8986dadb943db45b86cb2ddd00a187ea3380387b4d1dc242a97086a55c62e

COPY --from=builder /app/action /action

ENTRYPOINT ["/action"]
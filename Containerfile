FROM docker.io/golang:1.24-bookworm AS build

WORKDIR /src/
COPY go* .
COPY *.go .
RUN go build -v -o liaison

FROM docker.io/debian:bookworm-slim
RUN apt update
RUN apt install -y ca-certificates
COPY --from=build /src/liaison /liaison

ENTRYPOINT ["/liaison"]
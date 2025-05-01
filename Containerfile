FROM docker.io/golang:1.24-bookworm AS build

WORKDIR /src/
COPY go* .
COPY *.go .
RUN go build -v -o liason

FROM docker.io/debian:bookworm-slim
COPY --from=build /src/liason /liason

ENTRYPOINT ["/liason"]
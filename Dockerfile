FROM golang:1.17 AS build-go
ADD . /src
WORKDIR /src
RUN go build -o opaq .

FROM gcr.io/distroless/base
COPY --from=build-go /src/opaq /opaq
WORKDIR /
ENTRYPOINT ["/opaq"]

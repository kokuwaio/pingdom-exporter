FROM golang:1.23 AS build

WORKDIR /app
ADD . .
RUN make build

FROM alpine:3
MAINTAINER Daniel Martins <daniel.martins@jusbrasil.com.br>

COPY --from=build /app/bin/pingdom-exporter /pingdom-exporter
ENTRYPOINT ["/pingdom-exporter"]

USER 65534:65534

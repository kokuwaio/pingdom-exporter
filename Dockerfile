FROM golang:1.27 AS build

WORKDIR /app
COPY . .
RUN make build

FROM alpine:3

COPY --from=build /app/bin/pingdom-exporter /pingdom-exporter
ENTRYPOINT ["/pingdom-exporter"]

USER 65534:65534

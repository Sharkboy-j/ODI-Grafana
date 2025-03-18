FROM alpine:3.20
WORKDIR /
COPY odi-grafana /

CMD ["/odi-grafana"]
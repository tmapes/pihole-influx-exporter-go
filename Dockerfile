FROM golang:1.22.5-alpine@sha256:0d3653dd6f35159ec6e3d10263a42372f6f194c3dea0b35235d72aabde86486e

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]

FROM scratch

COPY pihole-influx-exporter /pihole-influx-exporter

CMD ["/pihole-influx-exporter"]
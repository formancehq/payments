FROM ghcr.io/formancehq/base:26.04
COPY payments /usr/bin/payments
ENV OTEL_SERVICE_NAME payments
ENTRYPOINT ["/usr/bin/payments"]
CMD ["server"]

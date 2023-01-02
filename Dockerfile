FROM alpine
ENTRYPOINT ["/usr/bin/gomake"]
COPY gomake /usr/bin/gomake
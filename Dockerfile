

FROM alpine 
COPY ./gomake /usr/bin/gomake
ENTRYPOINT [ "gomake" ]


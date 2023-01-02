
FROM golang as builder
RUN mkdir /gomake
COPY ./ /gomake/
WORKDIR /gomake
RUN ls -la . 
RUN go get
RUN go build

FROM alpine 
COPY --from=builder /gomake/gomake /usr/bin/gomake
ENTRYPOINT [ "gomake" ]


FROM scratch 
COPY gomake /gomake
ENTRYPOINT [ "/gomake" ]



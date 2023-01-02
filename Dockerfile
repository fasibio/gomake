FROM scratch 
RUN mkdir test
COPY . /test
RUN ls -la test
RUN ls -la .
COPY gomake /gomake
ENTRYPOINT [ "/gomake" ]



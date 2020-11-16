FROM ubuntu:focal

RUN apt-get update && apt-get install -y fio --no-install-recommends \
    && mkdir /fio

COPY test.fio /fio/
WORKDIR /fio 
CMD [ "fio", "test.fio" ]



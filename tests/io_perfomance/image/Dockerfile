FROM ubuntu:focal

RUN apt-get update && apt-get install -y fio=3.16-1 --no-install-recommends \
    && mkdir /fio

COPY test.fio /fio
WORKDIR /fio

CMD ["fio", "test.fio"]



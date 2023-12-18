#FROM ubuntu:22.04
FROM actlab.azurecr.io/repro_base

WORKDIR /app

ADD actlabs-hub ./

EXPOSE 8883/tcp

ENTRYPOINT [ "/bin/bash", "-c", "service redis-server start && ./actlabs-hub" ]
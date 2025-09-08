#FROM ubuntu:22.04
FROM actlabs.azurecr.io/actlabs-base:latest

WORKDIR /app

ADD actlabs-hub ./

EXPOSE 8883/tcp

ENTRYPOINT [ "/bin/bash", "-c", "service redis-server start && ./actlabs-hub" ]
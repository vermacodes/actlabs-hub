#FROM ubuntu:22.04
FROM actlabs.azurecr.io/actlabs-base:20251104-01

WORKDIR /app

ADD actlabs-hub ./

EXPOSE 8883/tcp

ENTRYPOINT [ "/bin/bash", "-c", "./actlabs-hub" ]
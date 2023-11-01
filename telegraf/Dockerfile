FROM telegraf:1.24

LABEL author="Volodymyr Kuchinskyi <kuchinskiy.wladimir@gmail.com>"
LABEL version="1.0.0"
LABEL description="Telegraf docker image"

USER root

RUN apt-get update && apt-get -y install build-essential hddtemp

ADD telegraf.conf.template /telegraf.conf.template
COPY *.conf /etc/telegraf/telegraf.d/

ADD run.sh /run.sh
ENTRYPOINT ["bash", "/run.sh"]
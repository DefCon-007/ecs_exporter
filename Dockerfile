ARG ARCH="amd64"
ARG OS="linux"

FROM grafana/agent:v0.34.3

RUN apt update && apt install -y ca-certificates

COPY ./grafana-agent-config.yml /etc/grafana-agent-config.yml
COPY ./ecs_exporter /bin/ecs_exporter
COPY ./entrypoint.sh /bin/entrypoint.sh

EXPOSE 9779

ENTRYPOINT ["sh", "/bin/entrypoint.sh" ]

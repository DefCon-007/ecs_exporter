ARG ARCH="amd64"
ARG OS="linux"

FROM golang:1.20-alpine as build

WORKDIR /app
RUN apk add --no-cache make curl git

COPY .promu.yml /app/
COPY go.mod /app/
COPY go.sum /app/
COPY Makefile /app/
COPY Makefile.common /app/

RUN make promu
RUN go mod download

COPY . /app/

RUN make build

FROM grafana/agent:v0.34.3

RUN apt update && apt install -y ca-certificates

COPY ./grafana-agent-config.yml /etc/grafana-agent-config.yml
COPY --from=build /app/ecs_exporter /bin/ecs_exporter
RUN chmod +x /bin/ecs_exporter
COPY ./entrypoint.sh /bin/entrypoint.sh

EXPOSE 9779

ENTRYPOINT ["sh", "/bin/entrypoint.sh" ]

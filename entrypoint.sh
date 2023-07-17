#!/bin/sh

# Start ECS exporter in background
/bin/ecs_exporter ${ECS_EXPORTER_CLI_ARGS} &

# Start grafana agent
/bin/grafana-agent --config.file=/etc/grafana-agent-config.yml --config.expand-env

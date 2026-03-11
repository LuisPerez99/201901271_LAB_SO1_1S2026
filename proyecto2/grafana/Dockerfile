FROM grafana/grafana:latest

RUN grafana-cli plugins install redis-datasource
VOLUME /var/lib/grafana
EXPOSE 3000

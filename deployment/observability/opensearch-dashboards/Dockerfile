FROM opensearchproject/opensearch-dashboards:2.11.1

RUN /usr/share/opensearch-dashboards/bin/opensearch-dashboards-plugin remove securityDashboards

COPY --chown=opensearch-dashboards:opensearch-dashboards opensearch_dashboards.yml /usr/share/opensearch-dashboards/config/opensearch_dashboards.yml

---
# Keycloak Custom Resource patch configuration
# Defines a highly available Keycloak deployment with PostgreSQL backend
apiVersion: k8s.keycloak.org/v2beta1
kind: Keycloak
metadata:
  name: keycloak
  namespace: keycloak
spec:
  # Deployment configuration
  startOptimized: {{ .OpenCenter.Services.keycloak.StartOptimized }}
  instances: {{ .OpenCenter.Services.keycloak.Instances }}
  
  # Resource limits for production
  resources:
    requests:
      memory: {{ .OpenCenter.Services.keycloak.ResourceRequestsMemory | default "1250M" }}
      cpu: {{ .OpenCenter.Services.keycloak.ResourceRequestsCPU }}
    limits:
      memory: {{ .OpenCenter.Services.keycloak.ResourceLimitsMemory | default "2250M" }}
      cpu: {{ .OpenCenter.Services.keycloak.ResourceLimitsCPU }}

  # Database configuration (PostgreSQL)
  db:
    vendor: postgres
    usernameSecret:
      name: keycloak.postgres-cluster.credentials.postgresql.acid.zalan.do
      key: username
    passwordSecret:
      name: keycloak.postgres-cluster.credentials.postgresql.acid.zalan.do
      key: password
    url: jdbc:postgresql://postgres-cluster.keycloak.svc.cluster.local:5432/keycloak
    poolMinSize: {{ .OpenCenter.Services.keycloak.DBPoolMinSize | default 30 }}
    poolInitialSize: {{ .OpenCenter.Services.keycloak.DBPoolInitialSize | default 30 }}
    poolMaxSize: {{ .OpenCenter.Services.keycloak.DBPoolMaxSize | default 30 }}

  # HTTP configuration
  http:
    httpEnabled: true
    {{- if .OpenCenter.Services.keycloak.TLSEnabled | default true }}
    tlsSecret: {{ .OpenCenter.Services.keycloak.TLSSecretName | default "keycloak-tls-secret" }}
    {{- end }}
  
  hostname:
    hostname: {{ .OpenCenter.Services.keycloak.Hostname | default (printf "auth.%s" .OpenCenter.Cluster.ClusterFQDN) }}
    strict: false
    backchannelDynamic: false
  
  proxy:
    headers: xforwarded
  
  additionalOptions:
    - name: proxy
      value: "edge"
    - name: hostname-url
      value: "https://{{ .OpenCenter.Services.keycloak.Hostname | default (printf "auth.%s" .OpenCenter.Cluster.ClusterFQDN) }}"
    {{- if .OpenCenter.Services.keycloak.MetricsEnabled | default true }}
    - name: metrics-enabled
      value: "true"
    {{- end }}
    {{- if .OpenCenter.Services.keycloak.EventMetricsEnabled | default true }}
    - name: event-metrics-user-enabled
      value: "true"
    {{- end }}
    {{- if .OpenCenter.Services.keycloak.HealthEnabled | default true }}
    - name: health-enabled
      value: "true"
    {{- end }}
    - name: log-level
      value: {{ .OpenCenter.Services.keycloak.LogLevel | default "INFO" | upper }}
    - name: log-console-output
      value: {{ .OpenCenter.Services.keycloak.LogFormat | default "json" }}
    {{- if .OpenCenter.Services.keycloak.CacheEnabled | default true }}
    - name: cache
      value: {{ .OpenCenter.Services.keycloak.CacheStack | default "ispn" }}
    {{- end }}
    - name: spi-connections-http-client-default-connection-timeout-millis
      value: "60000"
  {{- if ne .OpenCenter.Infrastructure.Provider "kind" }}
  # Pod topology spread for multi-AZ distribution (not needed for local Kind)
  unsupported:
    podTemplate:
      spec:
        topologySpreadConstraints:
          - maxSkew: 1
            topologyKey: "topology.kubernetes.io/zone"
            whenUnsatisfiable: "ScheduleAnyway"
            labelSelector:
              matchLabels:
                app: "keycloak"
                app.kubernetes.io/managed-by: "keycloak-operator"
                app.kubernetes.io/instance: "keycloak"
                app.kubernetes.io/component: "server"
          - maxSkew: 1
            topologyKey: "kubernetes.io/hostname"
            whenUnsatisfiable: "ScheduleAnyway"
            labelSelector:
              matchLabels:
                app: "keycloak"
                app.kubernetes.io/managed-by: "keycloak-operator"
                app.kubernetes.io/instance: "keycloak"
                app.kubernetes.io/component: "server"
  {{- end }}



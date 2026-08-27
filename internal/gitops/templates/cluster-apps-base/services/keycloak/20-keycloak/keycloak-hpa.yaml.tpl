{{- if and (.OpenCenter.Services.keycloak.MinReplicas) (.OpenCenter.Services.keycloak.MaxReplicas) }}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: keycloak-hpa
  namespace: keycloak
spec:
  scaleTargetRef:
    apiVersion: k8s.keycloak.org/v2beta1
    kind: Keycloak
    name: keycloak
  minReplicas: {{ .OpenCenter.Services.keycloak.MinReplicas | default 3 }}
  maxReplicas: {{ .OpenCenter.Services.keycloak.MaxReplicas | default 10 }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 85
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 100
          periodSeconds: 60
        - type: Pods
          value: 2
          periodSeconds: 60
      selectPolicy: Max
{{- end }}

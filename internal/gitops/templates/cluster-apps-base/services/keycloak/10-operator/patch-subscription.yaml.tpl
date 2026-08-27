apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: keycloak-operator
  namespace: keycloak
spec:
  name: keycloak-operator
  channel: fast
  source: operatorhubio-catalog
  sourceNamespace: olm
  startingCSV: keycloak-operator.v26.4.2
  installPlanApproval: Automatic

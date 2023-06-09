# generic-cluster-healthz

It exposes a http probe, by invoking this probe to check a cluster is healthy.
The probe handler checks ready nodes and ready non host networking pods.

This could be used with karmada, to make karmada checks member cluster's healthiness
accurately.

## example

```bash
# generate member rbac resources
helm template --namespace=kube-system member-cluster-rbac ./member-cluster-rbac | tee template.yaml
```

```powershell
helm template --namespace=kube-system member-cluster-rbac .\member-cluster-rbac | % ToString | Tee-Object template.yaml
```

```bash
# generate k8s resources for running the probe
# give it a name, to distinct from other member clusters
helm template --namespace=kube-system --set name=member-dev-healthz host-cluster-component ./host-cluster-component | tee host-template.yaml
```

```powershell
helm template --namespace=kube-system --set name=member-dev-healthz host-cluster-component .\host-cluster-component | % ToString | Tee-Object host-template.yaml
```

```yaml
# annotate the cluser cr
apiVersion: cluster.karmada.io/v1alpha1
kind: Cluster
metadata:
  annotations:
    cluster.karmada.io/health-checker: '{"service":{"namespace":"kube-system","name":"member-dev-healthz","port":8443,"path":"/cluster-healthz"},"caBundle":"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lRTjRFSGc1TU16YkFHdlJDbnRUMVpVekFOQmdrcWhraUc5dzBCQVFzRkFEQU4KTVFzd0NRWURWUVFERXdKallUQWVGdzB5TXpBMk1Ea3dNakkzTXpCYUZ3MHpNekEyTURZd01qSTNNekJhTUEweApDekFKQmdOVkJBTVRBbU5oTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0M1RtCmpHN1ZyNmVlWFlURnIrRlNQODZXSEtZQ0s4QnFLdkhHZjJXNERKODgwQW9TbEdTd3NsaktleW0ycmdseHVETysKODBUWEVpRmtrRHJOK093YnRBK0pqZWxoQ0Q2ME1YYS84Q3R4S3RBVE9XUFVZajkybTNhSnEyaXBBenJaY3plKwpVaDJnVGVJQzZvSkJhVjEzZ1BweW9XdmM2NmhIMXphYUZIOGNaVmxMRVJteC9XSEtvUEJBV3Fia3ZTUStCdFFQCmdsWGdLa1phMkJpVWM1Z3ZydGl0dU1HaEV4TFp0YVpXNTBVdy9JcnNaNUpEMmpvMHFBUm9ZdE4zb3k1d2dPa2oKWk8yWHhyMFErVVBHWnR6ckVPZGY4ekhseCs3VjBpc1pnV3o5YklYYWRsZjlUd0QwMWlxMDJjZ3RUUlgzMStMcAplVG5VQi8wREN5OERWUEFkMlFJREFRQUJvMkV3WHpBT0JnTlZIUThCQWY4RUJBTUNBcVF3SFFZRFZSMGxCQll3CkZBWUlLd1lCQlFVSEF3RUdDQ3NHQVFVRkJ3TUNNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUUKRk1zSHNtck1janNscUVqek5DdG5PMUI4ZkFtVk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRRENZbmNPdDJ0bQpRUjZTVjltWjZ1VklENnREcTN2cHJqNm5PUjFZZ2tibXpRZStSWlZSRkJGZkUxMW0vbVdwdE1QeWtkSythVUo1Cm1WcCtiQkE1dTlDV1hXTXlONXZla2JETVFWS3BtTUUvYXltMEJkZGpjQmpJTFUvZllKcnRkU3Q1VjZkcEJ5U3oKdkVDYWdFRU5YcEEwQ1lRb1ZuM2I4L1oxNllHRmFYa3VxalVBY3MzK1lYWmZpQ0NVUzVFTUZiWmlJNkwwdEU3YQpWM0RUWmorVFEzWnAwb0FMRW1HaE9kSHprNS8vOUxxYXBYTkZjRHIvTFdVTXdqMFRQUS9TZm5ZbFlQN2h2dmgzCkhNOUVjZzEyUHRnNStlTElzS1FMbnZGcXNLNTdJcWlBbUNiaS9DSGt4cGh2bVhUYVA5dDYwRkJKd3ZrSUlNaDgKcG5obDBhdEo5QThXCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"}'
```
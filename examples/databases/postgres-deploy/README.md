# How to deploy a simple Postgres DB to Kubernetes cluster

This is a very simple example to show how to create a Postgres database. **Note**, this is not ready for any production purpose.

## Create a Kubernetes Deployment
```
kubectl create -f postgres-configmap.yaml
kubectl create -f postgres-storage.yaml
kubectl create -f postgres-deployment.yaml
kubectl create -f postgres-service.yaml
```
## Test the connection

Connection credentials available in the _postgres-configmap.yaml_ descriptor.

```
kubectl get svc postgres
psql -h <IP> -U postgresadmin --password -p <PORT> postgresdb
```
## Create a test database and table
```
CREATE DATABASE test;
CREATE TABLE test (data TEXT PRIMARY KEY);
INSERT INTO test(data) VALUES ('hello'), ('world');
```
### Read the test database and table
```
SELECT * FROM test;
```


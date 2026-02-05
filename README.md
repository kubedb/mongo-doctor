# mongo-doctor


### Run
```bash
Edit the yamls/job.yaml file to set the ENVs

make
```

### Share the information

```bash
kubectl logs -n demo job/doctor -f

# You will be notified when to run `kubectl cp`
# For copying the output of stats commands from pod
kubectl cp demo/<doctor-pod>:/all /tmp/data 
```


### URL
mongo --tls --tlsCAFile /var/run/mongodb/tls/ca.crt --tlsCertificateKeyFile /var/run/mongodb/tls/client.pem admin --host localhost --authenticationMechanism MONGODB-X509 --quiet
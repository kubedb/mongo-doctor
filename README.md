# mongo-doctor


### Run
```bash
Edit the yamls/pod.yaml file to set the ENVs

make
```

### Share the information

```bash
kubectl logs -n demo job/doctor -f

# You will be notified when to run `kubectl cp`
# For copying the output of stats commands from pod
kubectl cp demo/<doctor-pod>:/app/all-stats /tmp/data 
```

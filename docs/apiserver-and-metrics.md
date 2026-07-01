# Control Plane Metrics Configuration

## K3s Clusters   
                  
On K3s, depending on the exact K3s version by default the `kube-apiserver`, `kube-controller-manager` and `kube-scheduler`,
ports might be tied to the localhost `127.0.0.1` only which is insufficient for accessing it from within the cluster.
Ensure that your `k3s-server` processes are started with the following arguments:

```bash
--kube-apiserver-arg=bind-address=0.0.0.0 \                                                                                                                                                                                                                                                   
--kube-controller-manager-arg=bind-address=0.0.0.0 \                                                                                                                                                                                                                                          
--kube-scheduler-arg=bind-address=0.0.0.0 \
```

Furthermore the `kube-controller-manager` and `kube-scheduler` services from the `kube-system` might be missing by default:


## RKE2 Cluster

On RKE2, depending on the exact RKE2 version by default the `kube-apiserver`, `kube-controller-manager` and `kube-scheduler`,
ports might be tied to the localhost `127.0.0.1` only which is insufficient for accessing it from within the cluster.
Ensure that your `/etc/rancher/rke2/config.yaml` file has the following:

```yaml
etcd-arg:                                                                                                                                                                                                                                                                                       
  - "listen-metrics-urls=http://0.0.0.0:2381"                                                                                                                                                                                                                                                   
kube-scheduler-arg:                                                                                                                                                                                                                                                                             
  - "bind-address=0.0.0.0"                                                                                                                                                                                                                                                                      
kube-controller-manager-arg:                                                                                                                                                                                                                                                                    
  - "bind-address=0.0.0.0"                                                                                                                                                                                                                                                                      
kube-apiserver-arg:                                                                                                                                                                                                                                                                             
  - "bind-address=0.0.0.0"
```

This should be present, either before cluster installation, or if you add it after, then make sure you restart the server nodes with:

```bash
sudo systemctl restart rke2-server.service
```

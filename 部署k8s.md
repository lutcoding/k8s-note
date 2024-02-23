# 部署Kubernetes集群

## 所有主机上执行

```yaml
# 集群信息
192.168.246.135 k8s-master
192.168.246.136 k8s-node1
192.168.246.137 k8s-node2

# 在各自主机上修改主机名
hostnamectl set-hostname k8s-master
hostnamectl set-hostname k8s-node1
hostnamectl set-hostname k8s-node2

# 配置hosts
gedit /etc/hosts
将以下信息添加到每台主机的hosts文件
192.168.246.135 k8s-master
192.168.246.136 k8s-node1
192.168.246.137 k8s-node2

# 关闭swap交换分区
swapoff -a
## 注释掉UUID=1d4a68d0-df0f-43f1-b126-ea43efa1a778 none            swap    sw              0       0  行
gedit /etc/fstab

# 关闭防火墙
ufw disable
ufw status

sudo apt-get install -y bridge-utils
sudo modprobe br_netfilter

# 安装docker
apt install docker.io -y

# 配置镜像仓库
cat > /etc/docker/daemon.json <<EOF
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "registry-mirrors": ["https://docker.mirrors.ustc.edu.cn"]
}
EOF
# 重启docker
systemctl daemon-reload && systemctl restart docker

#安装k8s
sudo apt-get install -y ca-certificates curl software-properties-common apt-transport-https
curl -s https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg | sudo apt-key add -
sudo tee /etc/apt/sources.list.d/kubernetes.list <<EOF
deb https://mirrors.aliyun.com/kubernetes/apt/ kubernetes-xenial main
EOF
sudo apt-get update
# 查看可用版本
apt-cache madison kubeadm
sudo apt-get install -y kubelet=1.23.5-00 kubeadm=1.23.5-00 kubectl=1.23.5-00
# 锁定版本
sudo apt-mark hold kubelet kubeadm kubectl
```

## 部署control-plane节点

```
sudo kubeadm init --config kubeadm-config.yaml
```

### kubeadm-config.yaml

```yaml
apiVersion: kubeadm.k8s.io/v1beta3
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.0123456789abcdef
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.246.135
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  imagePullPolicy: IfNotPresent
  name: k8s-master
  taints: null
---
apiServer:
  timeoutForControlPlane: 4m0s
apiVersion: kubeadm.k8s.io/v1beta3
certificatesDir: /etc/kubernetes/pki
clusterName: kubernetes
controllerManager: {}
dns: {}
etcd:
  local:
    dataDir: /var/lib/etcd
imageRepository: registry.cn-hangzhou.aliyuncs.com/google_containers
kind: ClusterConfiguration
kubernetesVersion: 1.23.5
networking:
  dnsDomain: cluster.local
  podSubnet: 172.16.0.0/16
  serviceSubnet: 10.96.0.0/12
scheduler: {}
```

可以通过`kubeadm config print init-defaults`打印默认yaml配置

可以看到

```shell
Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 192.168.246.135:6443 --token abcdef.0123456789abcdef \
	--discovery-token-ca-cert-hash sha256:3efebbb4cb418dc32b6adaf88a5ecb0a7084b151dd9346a2cf4fab68228d0d44 
```

`kubeadm init` 完成后，提示需要做三件事情：
a.配置环境变量
b.安装网络插件
c.将节点加入集群

#### a.配置环境变量

```shell
## 非root下执行
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# 切换到root执行
export KUBECONFIG=/etc/kubernetes/admin.conf
```

#### b.安装网络插件

允许`kubectl get node`可以看到`NotReady`，因为没有安装网络插件

```shell
wildhunt@k8s-master:~$ kubectl get node
NAME         STATUS     ROLES                  AGE     VERSION
k8s-master   NotReady   control-plane,master   8m49s   v1.23.5
```

```shell
kubectl apply -f https://docs.projectcalico.org/v3.21/manifests/calico.yaml

## 可以看到master节点已经Ready了
wildhunt@k8s-master:~$ kubectl get node
NAME         STATUS   ROLES                  AGE   VERSION
k8s-master   Ready    control-plane,master   14m   v1.23.5
```

#### c.将节点加入集群(在worker节点执行)

```shell
# 加入集群
kubeadm join 192.168.246.135:6443 --token abcdef.0123456789abcdef \
	--discovery-token-ca-cert-hash sha256:3efebbb4cb418dc32b6adaf88a5ecb0a7084b151dd9346a2cf4fab68228d0d44 
	
# 如果token过期,可以通过下面命令获取
kubeadm token create --print-join-command
```

## 安装**MetalLB**

MetalLB 是一个用于裸机 Kubernetes 集群的负载均衡器实现，使用标准路由协议。

k8s 并没有为裸机集群实现负载均衡器，因此我们只有在以下 IaaS 平台（GCP, AWS, Azure）上才能使用 LoadBalancer 类型的 service。

因此裸机集群只能使用 NodePort 或者 externalIPs service 来对面暴露服务，然而这两种方式和 LoadBalancer service 相比都有很大的缺点。

而 MetalLB 的出现就是为了解决这个问题。

### 安装 MetalLB

```shell
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
```

### 配置Layer 2 模式配置

```yaml
cat <<EOF > IPAddressPool.yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: first-pool
  namespace: metallb-system
spec:
  addresses:
  # 可分配的 IP 地址,可以指定多个，包括 ipv4、ipv6
  # 必须与虚拟机同一网段，这样宿主机才能访问分配的IP
  # 虚拟机192.168.246.0/24 子网掩码255.255.255.0
  - 192.168.246.220-192.168.246.254
EOF

kubectl apply -f IPAddressPool.yaml
```

```yaml
cat <<EOF > L2Advertisement.yaml
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: example
  namespace: metallb-system
spec:
  ipAddressPools:
  - first-pool #上一步创建的 ip 地址池，通过名字进行关联
EOF

kubectl apply -f L2Advertisement.yaml
```


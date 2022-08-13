#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -e

# This script runs e2e test against on kubean control plane.
# You should prepare your environment in advance and following environment may be you need to set or use default one.
# - CONTROL_PLANE_KUBECONFIG: absolute path of control plane KUBECONFIG file.
#
# Usage: hack/run-e2e.sh

# Run e2e 
KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
HOST_CLUSTER_NAME=${1:-"kubean-host"}
SPRAY_JOB_VERSION=${2:-latest}
vm_ip_addr1=${3:-"10.6.127.33"}
vm_ip_addr2=${4:-"10.6.127.33"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/${HOST_CLUSTER_NAME}.config"}
EXIT_CODE=0
echo "==> currnent dir: "$(pwd)
# Install ginkgo
GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin

# prepare vagrant vm as k8 cluster single node
vm_clean_up(){
    vagrant destroy -f default
    exit $EXIT_CODE
}

#trap vm_clean_up EXIT
# create single node for cluster
sed -i "s/default_ip/${vm_ip_addr1}/" Vagrantfile
sed -i "s/default2_ip/${vm_ip_addr2}/" Vagrantfile
vagrant up
vagrant status
ATTEMPTS=0
pingOK=0
ping -w 2 -c 1 $vm_ip_addr1|grep "0%" && pingOK=true || pingOK=false
until [ "${pingOK}" == "false" ] || [ $ATTEMPTS -eq 10 ]; do
ping -w 2 -c 1 $vm_ip_addr1|grep "0%" && pingOK=true || pingOK=false
echo "==> ping "$vm_ip_addr1 $pingOK
ATTEMPTS=$((ATTEMPTS + 1))
sleep 10
done

sshpass -p root ssh root@${vm_ip_addr1} cat /proc/version
ping -c 5 ${vm_ip_addr2}
# print vm origin hostname
echo "==> before deploy display single node hostname: "
sshpass -p root ssh root@${vm_ip_addr1} hostname
echo "==> scp sonobuoy to master: "
sshpass -p root scp $(pwd)/test/tools/sonobuoy_0.56.9_linux_amd64.tar.gz root@$vm_ip_addr1:/root/

# prepare kubean install job yml using containerd
SPRAY_JOB="ghcr.io/kubean-io/kubean/spray-job:${SPRAY_JOB_VERSION}"
cp $(pwd)/test/common/* $(pwd)/test/kubean_functions_e2e/e2e-install-cluster/
sed -i "s/vm_ip_addr1/${vm_ip_addr1}/" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster/hosts-conf-cm.yml
sed -i "s/vm_ip_addr2/${vm_ip_addr2}/" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster/hosts-conf-cm.yml
sed -i "s#image:#image: ${SPRAY_JOB}#" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster/kubeanClusterOps.yml

# prepare kubean reset job yml
cp $(pwd)/test/common/* $(pwd)/test/kubean_functions_e2e/e2e-reset-cluster/
sed -i "s/vm_ip_addr1/${vm_ip_addr1}/" $(pwd)/test/kubean_functions_e2e/e2e-reset-cluster/hosts-conf-cm.yml
sed -i "s/vm_ip_addr2/${vm_ip_addr2}/" $(pwd)/test/kubean_functions_e2e/e2e-reset-cluster/hosts-conf-cm.yml
sed -i "s#image:#image: ${SPRAY_JOB}#" $(pwd)/test/kubean_functions_e2e/e2e-reset-cluster/kubeanClusterOps.yml

# prepare kubean install job yml using docker
cp -r $(pwd)/test/kubean_functions_e2e/e2e-install-cluster $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker
sed -i "s#e2e-cluster1-install#e2e-install-cluster-docker#" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker/kubeanClusterOps.yml
sed -i "s/containerd/docker/" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker/vars-conf-cm.yml
sed -i "s#  \"10.6.170.10:5000\": \"http://10.6.170.10:5000\"#   - 10.6.170.10:5000#" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker/vars-conf-cm.yml
# TBD: kube_network_plugin=cillium; cause' the core version of centos79 is low 3.10.0, cillium require high core version more than 4.x; so such case id pending.
# sed -i "s#kube_network_plugin: calico#kube_network_plugin: cillium#" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker/vars-conf-cm.yml
# override_system_hostname=false
sed -i "$ a\    override_system_hostname: false" $(pwd)/test/kubean_functions_e2e/e2e-install-cluster-docker/vars-conf-cm.yml

# prepare kubean ops yml
cp $(pwd)/test/common/kubeanCluster.yml $(pwd)/test/kubeanOps_functions_e2e/e2e-install-cluster/
cp $(pwd)/test/common/vars-conf-cm.yml $(pwd)/test/kubeanOps_functions_e2e/e2e-install-cluster/

# Run nightly e2e
ginkgo -v -race --fail-fast ./test/kubean_deploy_e2e/  -- --kubeconfig="${MAIN_KUBECONFIG}"
ginkgo -v -race --fail-fast ./test/kubean_functions_e2e/  -- --kubeconfig="${MAIN_KUBECONFIG}" --vmipaddr="${vm_ip_addr1}"
# ginkgo -v -race --fail-fast ./test/kubeanOps_functions_e2e/  -- --kubeconfig="${MAIN_KUBECONFIG}" --vmipaddr="${vm_ip_addr1}"

#!/usr/bin/env bash
set -o nounset
set -o pipefail
set -e
set -x

TARGET_VERSION=${1:-v0.0.0}
IMAGE_VERSION=${2:-latest}
HELM_REPO=${3:-"https://kubean-io.github.io/kubean-helm-chart"}
IMG_REPO=${4:-"ghcr.io/kubean-io"}
SPRAY_JOB_VERSION=${5:-latest}
RUNNER_NAME=${6:-"kubean-actions-runner1"}
EXIT_CODE=0

CLUSTER_PREFIX=kubean-"${IMAGE_VERSION}"-$RANDOM
REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${REPO_ROOT}"/hack/e2e.sh

local_helm_repo_alias="kubean_release"
# add kubean repo locally
repoCount=$(helm repo list | grep "${local_helm_repo_alias}" && repoCount=true || repoCount=false)
if [ "$repoCount" == "true" ]; then
    helm repo remove ${local_helm_repo_alias}
else
    echo "repoCount:" $repoCount
fi
helm repo add ${local_helm_repo_alias} ${HELM_REPO}
helm repo update
helm repo list

chmod +x ./hack/delete-cluster.sh
chmod +x ./hack/local-up-kindcluster.sh

# prepare vagrant vm as k8 cluster single node
vm_clean_up(){
    vagrant destroy -f default
    exit $EXIT_CODE
}

install_sshpass(){
    local CMD=$(command -v ${1})
    if [[ ! -x ${CMD} ]]; then
        echo "Installing sshpass: "
        wget http://sourceforge.net/projects/sshpass/files/sshpass/1.05/sshpass-1.05.tar.gz
        tar xvzf sshpass-1.05.tar.gz
        cd sshpass-1.05.tar.gz
        ./configure
        make
        make install
    fi
}

os_compitable_e2e(){
    KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
    HOST_CLUSTER_NAME=${1:-"kubean-host"}
    #SPRAY_JOB_VERSION=${2:-latest}
    #vm_ip_addr=${3:-"10.6.127.33"}
    MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/${HOST_CLUSTER_NAME}.config"}
    EXIT_CODE=0
    echo "currnent dir: "$(pwd)
    # Install ginkgo
    GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
    export PATH=$PATH:$GOPATH/bin

    trap vm_clean_up EXIT
    #prepare master vm
    vagrant init Kiowa/kkubean-redhat84 --box-version 0
    sed -i "$ i\  config.vm.network \"public_network\", ip: \"${vm_ip_addr1}\", bridge: \"ens192\"" Vagrantfile
    vagrant up
    vagrant status
    ping -c 5 ${vm_ip_addr1}
    sshpass -p root ssh -o StrictHostKeyChecking=no  root@${vm_ip_addr1} cat /proc/version
    # prepare kubean install job yml using containerd
    SPRAY_JOB="ghcr.io/kubean-io/spray-job:${SPRAY_JOB_VERSION}"
    cp $(pwd)/test/common/hosts-conf-cm.yml $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/
    cp $(pwd)/test/common/kubeanCluster.yml $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/
    cp $(pwd)/test/common/vars-conf-cm.yml $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/
    sed -i "s/ip:/ip: ${vm_ip_addr}/" $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/hosts-conf-cm.yml
    sed -i "s/ansible_host:/ansible_host: ${vm_ip_addr}/" $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/hosts-conf-cm.yml
    sed -i "s#image:#image: ${SPRAY_JOB}#" $(pwd)/test/kubean_oscompitable_e2e/e2e-install-cluster/kubeanClusterOps.yml
    # Run cluster function e2e
    ginkgo -v -race --fail-fast ./test/kubean_oscompitable_e2e/  -- --kubeconfig="${MAIN_KUBECONFIG}"
}


###### OS compitable e2e logic ########
trap utils::clean_up EXIT
./hack/local-up-kindcluster.sh "${TARGET_VERSION}" "${IMAGE_VERSION}" "${HELM_REPO}" "${IMG_REPO}" "kindest/node:v1.21.1" "${CLUSTER_PREFIX}"-host
utils:runner_ip
install_sshpass
os_compitable_e2e

ret=$?
if [ ${ret} -ne 0 ]; then
  EXIT_CODE=1
fi




package kubean_calico_e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubean-io/kubean/test/tools"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubeanClusterClientSet "kubean.io/api/generated/kubeancluster/clientset/versioned"
)

var _ = ginkgo.Describe("Calico single stack tunnel: IPIP_ALWAYS", func() {

	config, err := clientcmd.BuildConfigFromFlags("", tools.Kubeconfig)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
	kubeClient, err := kubernetes.NewForConfig(config)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
	localKubeConfigPath := "calico-single-stack.config"
	var masterSSH = fmt.Sprintf("root@%s", tools.Vmipaddr)
	var workerSSH = fmt.Sprintf("root@%s", tools.Vmipaddr2)

	defer ginkgo.GinkgoRecover()

	ginkgo.Context("when install a cluster based on calico single stack", func() {
		clusterInstallYamlsPath := "e2e-install-calico-cluster"
		kubeanNamespace := "kubean-system"
		kubeanClusterOpsName := "e2e-install-calico-cluster"

		// firstly: apply vars-conf-cm
		tools.CreatVarsCM()
		// Create yaml for kuBean CR and related configuration
		installYamlPath := fmt.Sprint(tools.GetKuBeanPath(), clusterInstallYamlsPath)
		// do cluster deploy in containerd mode
		cmd := exec.Command("kubectl", "--kubeconfig="+tools.Kubeconfig, "apply", "-f", installYamlPath)
		ginkgo.GinkgoWriter.Printf("cmd: %s\n", cmd.String())
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			ginkgo.GinkgoWriter.Printf("apply cmd error: %s\n", err.Error())
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), stderr.String())
		}

		// Check if the job and related pods have been created
		time.Sleep(30 * time.Second)
		pods, _ := kubeClient.CoreV1().Pods(kubeanNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=kubean-%s-job", kubeanClusterOpsName),
		})
		gomega.Expect(len(pods.Items)).NotTo(gomega.Equal(0))
		jobPodName := pods.Items[0].Name

		// Wait for kubean job-related pod status to be succeeded
		for {
			pod, err := kubeClient.CoreV1().Pods(kubeanNamespace).Get(context.Background(), jobPodName, metav1.GetOptions{})
			ginkgo.GinkgoWriter.Printf("* wait for install job related pod[%s] status: %s\n", pod.Name, pod.Status.Phase)
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed get job related pod")
			podStatus := string(pod.Status.Phase)
			if podStatus == "Succeeded" || podStatus == "Failed" {
				ginkgo.It("kubean cluster podStatus should be Succeeded", func() {
					gomega.Expect(podStatus).To(gomega.Equal("Succeeded"))
				})
				break
			}
			time.Sleep(1 * time.Minute)
		}

		clusterClientSet, err := kubeanClusterClientSet.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")

		// from KuBeanCluster: cluster1 get kubeconfRef: name: cluster1-kubeconf namespace: kubean-system
		cluster1, err := clusterClientSet.KubeanV1alpha1().KuBeanClusters().Get(context.Background(), "cluster1", metav1.GetOptions{})
		fmt.Println("Name:", cluster1.Spec.KubeConfRef.Name, "NameSpace:", cluster1.Spec.KubeConfRef.NameSpace)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to get KuBeanCluster")

		// get configmap
		kubeClient, err := kubernetes.NewForConfig(config)
		cluster1CF, err := kubeClient.CoreV1().ConfigMaps(cluster1.Spec.KubeConfRef.NameSpace).Get(context.Background(), cluster1.Spec.KubeConfRef.Name, metav1.GetOptions{})
		err1 := os.WriteFile(localKubeConfigPath, []byte(cluster1CF.Data["config"]), 0666)
		gomega.ExpectWithOffset(2, err1).NotTo(gomega.HaveOccurred(), "failed to write localKubeConfigPath")

	})

	// check kube-system pod status
	ginkgo.Context("When fetching kube-system pods status", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")

		podList, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to check kube-system pod status")
		ginkgo.It("every pod in kube-system should be in running status", func() {
			for _, pod := range podList.Items {
				fmt.Println(pod.Name, string(pod.Status.Phase))
				gomega.Expect(string(pod.Status.Phase)).To(gomega.Equal("Running"))
			}
		})

	})

	//
	ginkgo.Context("Support CNI: Calico", func() {
		//4. check calico (calico-node and calico-kube-controller)pod status: pod status should be "Running"
		config, _ = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		kubeClient, _ = kubernetes.NewForConfig(config)
		podList, _ := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		for _, pod := range podList.Items {
			if strings.Contains(pod.ObjectMeta.Name, "calico-node") || strings.Contains(pod.ObjectMeta.Name, "kube-controller") {
				ginkgo.It("calico/controller pod should works", func() {
					gomega.Expect(string(pod.Status.Phase)).To(gomega.Equal("Running"))
				})
			}
		}

		//5. check folder /opt/cni/bin contains  file "calico" and "calico-ipam" are exist in both master and worker node
		masterCmd := tools.RemoteSSHCmdArray([]string{masterSSH, "ls", "/opt/cni/bin/"})
		workerCmd := tools.RemoteSSHCmdArray([]string{workerSSH, "ls", "/opt/cni/bin/"})
		out1, _ := tools.NewDoCmd("sshpass", masterCmd...)
		fmt.Println("out1: ", out1.String())
		ginkgo.It("master /opt/cni/bin checking: ", func() {
			gomega.Expect(out1.String()).Should(gomega.ContainSubstring("calico"))
		})
		out2, _ := tools.NewDoCmd("sshpass", workerCmd...)
		fmt.Println("out2: ", out2.String())
		ginkgo.It("worker /opt/cni/bin checking: ", func() {
			gomega.Expect(out2.String()).Should(gomega.ContainSubstring("calico"))
		})

		// check calicoctl
		masterCmd = tools.RemoteSSHCmdArray([]string{masterSSH, "calicoctl", "version"})
		out3, _ := tools.NewDoCmd("sshpass", masterCmd...)
		fmt.Println("out3: ", out3.String())
		ginkgo.It("master calicoctl checking: ", func() {
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("Client Version"))
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("Cluster Version"))
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("kubespray,kubeadm,kdd"))
		})

		//6. check pod connection:
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		//6.1. create a deployment of nginx1 on master, on namespace ns1: set replicaset to 1(here call the pod as pod1)
		nginx1Cmd := exec.Command("kubectl", "run", "nginx1", "-n", "kube-system", "--image", "nginx:alpine", "--kubeconfig", localKubeConfigPath, "--env", "NodeName=node1")
		nginx1CmdOut, err1 := tools.DoErrCmd(*nginx1Cmd)
		fmt.Println("create nginx1: ", nginx1CmdOut.String(), err1.String())
		nginx2Cmd := exec.Command("kubectl", "run", "nginx2", "-n", "default", "--image", "nginx:alpine", "--kubeconfig", localKubeConfigPath, "--env", "NodeName=node2")
		nginx2CmdOut, err2 := tools.DoErrCmd(*nginx2Cmd)
		fmt.Println("create nginx1: ", nginx2CmdOut.String(), err2.String())

		time.Sleep(60 * time.Second)
		pod1, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), "nginx1", metav1.GetOptions{})
		nginx1Ip := string(pod1.Status.PodIP)
		ginkgo.It("nginxPod1 should be in running status", func() {
			gomega.Expect(string(pod1.Status.Phase)).To(gomega.Equal("Running"))
		})
		pod2, _ := kubeClient.CoreV1().Pods("default").Get(context.Background(), "nginx2", metav1.GetOptions{})
		nginx2Ip := string(pod2.Status.PodIP)
		ginkgo.It("nginxPod1 should be in running status", func() {
			gomega.Expect(string(pod2.Status.Phase)).To(gomega.Equal("Running"))
		})
		// 4.1 node ping 2 pods
		pingNginx1IpCmd1 := tools.RemoteSSHCmdArray([]string{masterSSH, "ping", "-c 1", nginx1Ip})
		pingNginx1IpCmd1Out, _ := tools.NewDoCmd("sshpass", pingNginx1IpCmd1...)
		fmt.Println("node ping nginx pod 1: ", pingNginx1IpCmd1Out.String())
		ginkgo.It("node ping nginx pod 1 succuss: ", func() {
			gomega.Expect(pingNginx1IpCmd1Out.String()).Should(gomega.ContainSubstring("1 received"))
		})
		pingNginx2IpCmd1 := tools.RemoteSSHCmdArray([]string{masterSSH, "ping", "-c 1", nginx2Ip})
		pingNgin21IpCmd1Out, _ := tools.NewDoCmd("sshpass", pingNginx2IpCmd1...)
		fmt.Println("node ping nginx pod 2: ", pingNgin21IpCmd1Out.String())
		ginkgo.It("node ping nginx pod 2 succuss: ", func() {
			gomega.Expect(pingNgin21IpCmd1Out.String()).Should(gomega.ContainSubstring("1 received"))
		})
		// 4.2 pod ping pod
		podsPingCmd1 := tools.RemoteSSHCmdArray([]string{masterSSH, "kubectl", "exec", "-it", "nginx1", "-n", "kube-system", "--", "ping", "-c 1", nginx2Ip})
		podsPingCmdOut1, _ := tools.NewDoCmd("sshpass", podsPingCmd1...)
		fmt.Println("pod ping pod: ", podsPingCmdOut1.String())
		ginkgo.It("pod ping pod succuss: ", func() {
			gomega.Expect(podsPingCmdOut1.String()).Should(gomega.ContainSubstring("1 packets received"))
		})
		podsPingCmd2 := tools.RemoteSSHCmdArray([]string{masterSSH, "kubectl", "exec", "-it", "nginx2", "-n", "default", "--", "ping", "-c 1", nginx1Ip})
		podsPingCmdOut2, _ := tools.NewDoCmd("sshpass", podsPingCmd2...)
		fmt.Println("pod ping pod: ", podsPingCmdOut2.String())
		ginkgo.It("pod ping pod succuss: ", func() {
			gomega.Expect(podsPingCmdOut2.String()).Should(gomega.ContainSubstring("1 packets received"))
		})
	})

})

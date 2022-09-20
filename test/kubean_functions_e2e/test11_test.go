package kubeanOps_functions_e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kubean-io/kubean/test/tools"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	kubeanClusterClientSet "kubean.io/api/generated/kubeancluster/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var sshCmdArray = []string{"-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no"}

func RemoteSSHCmdArray(subCmd []string) []string {
	return append(sshCmdArray, subCmd...)
}

var _ = ginkgo.Describe("e2e test cluster operation", func() {

	config, err := clientcmd.BuildConfigFromFlags("", tools.Kubeconfig)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
	kubeClient, err := kubernetes.NewForConfig(config)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
	localKubeConfigPath := "cluster1-config"

	defer ginkgo.GinkgoRecover()

	ginkgo.Context("Containerd: when install a cluster", func() {
		clusterInstallYamlsPath := "e2e-install-cluster"
		kubeanNamespace := "kubean-system"
		kubeanClusterOpsName := "e2e-cluster1-install"

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
				ginkgo.It("kubean containerd cluster podStatus should be Succeeded", func() {
					gomega.Expect(podStatus).To(gomega.Equal("Succeeded"))
				})
				break
			}
			time.Sleep(1 * time.Minute)
		}

		clusterClientSet, err := kubeanClusterClientSet.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")

		// from KuBeanCluster: cluster1 get kubeconfRef: name: cluster1-kubeconf namespace: kubean-system
		cluster1, err := clusterClientSet.KubeanclusterV1alpha1().KuBeanClusters().Get(context.Background(), "cluster1", metav1.GetOptions{})
		fmt.Println("Name:", cluster1.Spec.KubeConfRef.Name, "NameSpace:", cluster1.Spec.KubeConfRef.NameSpace)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to get KuBeanCluster")

		// get configmap
		kubeClient, err := kubernetes.NewForConfig(config)
		cluster1CF, err := kubeClient.CoreV1().ConfigMaps(cluster1.Spec.KubeConfRef.NameSpace).Get(context.Background(), cluster1.Spec.KubeConfRef.Name, metav1.GetOptions{})
		err1 := os.WriteFile(localKubeConfigPath, []byte(cluster1CF.Data["config"]), 0666)
		gomega.ExpectWithOffset(2, err1).NotTo(gomega.HaveOccurred(), "failed to write localKubeConfigPath")

	})

	ginkgo.Context("when install nginx service", func() {
		fmt.Println("111111: ", localKubeConfigPath)
		config, err := clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err := kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		args := []string{"run", "nginx", "-n", "kube-system", "--image", "nginx:alpine", "--kubeconfig", localKubeConfigPath, "--env", "NodeName=node1"}
		nginx1Cmd := exec.Command("kubectl", args...)
		nginx1CmdOut, err1 := tools.DoErrCmd(*nginx1Cmd)
		fmt.Println("create nginx1: ", nginx1CmdOut.String(), err1.String())
		time.Sleep(60 * time.Second)

		masterSSH := fmt.Sprintf("root@%s", tools.Vmipaddr)
		masterCmd := RemoteSSHCmdArray([]string{masterSSH, "cat", "/proc/sys/net/ipv4/ip_forward"})
		//args := []string{"-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no"}
		//masterCmd := exec.Command("sshpass", "-p", "root", "ssh", masterSSH, "cat", "/proc/sys/net/ipv4/ip_forward", args...)
		fmt.Println("new masterCmd: ", masterCmd)
		// out1, _ := tools.DoCmd(*masterCmd)
		// fmt.Println("out: ", out1.String())
		cmd := exec.Command("sshpass", masterCmd...)
		ginkgo.GinkgoWriter.Printf("cmd: %s\n", cmd.String())
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			ginkgo.GinkgoWriter.Printf("apply cmd error: %s\n", err.Error())
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), stderr.String())
		}

		ginkgo.It("check pod ip is in kube_pods_subnet", func() {
			//the pod set was 192.168.128.0/20, so the available pod ip range is 192.168.128.1 ~ 192.168.143.255
			name := "nginx"
			pod, err := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), name, metav1.GetOptions{})
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to get pod")
			fmt.Println("222: ", pod.Status, pod.Status.PodIP)
			ipSplitArr := strings.Split(string(pod.Status.PodIP), ".")
			gomega.Expect(len(ipSplitArr)).Should(gomega.Equal(4))

			ipSub1, err := strconv.Atoi(ipSplitArr[0])
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "ip split conversion failed")
			ipSub2, err := strconv.Atoi(ipSplitArr[1])
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "ip split conversion failed")
			ipSub3, err := strconv.Atoi(ipSplitArr[2])
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "ip split conversion failed")

			gomega.Expect(ipSub1).Should(gomega.Equal(192))
			gomega.Expect(ipSub2).Should(gomega.Equal(168))
			gomega.Expect(ipSub3 >= 128).Should(gomega.BeTrue())
			gomega.Expect(ipSub3 <= 143).Should(gomega.BeTrue())

		})
	})

})

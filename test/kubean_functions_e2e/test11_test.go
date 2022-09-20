package kubeanOps_functions_e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kubean-io/kubean/test/tools"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = ginkgo.Describe("e2e test cluster operation", func() {

	localKubeConfigPath := "cluster1-sonobouy-config"
	localKubeConfigPath = fmt.Sprint(tools.GetKuBeanPath(), localKubeConfigPath)

	defer ginkgo.GinkgoRecover()

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
		args := []string{"-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no"}
		masterCmd := exec.Command("sshpass", "-p", "root", "ssh", masterSSH, "cat", "/proc/sys/net/ipv4/ip_forward", args...)
		out1, _ := tools.DoCmd(*masterCmd)
		fmt.Println("out: ", out1.String())

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

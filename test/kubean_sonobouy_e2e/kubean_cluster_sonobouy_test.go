package kubean_sonobouy_e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/kubean-io/kubean/test/tools"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubeanClusterClientSet "kubean.io/api/generated/kubeancluster/clientset/versioned"
)

var _ = ginkgo.Describe("e2e test cluster 1 master + 1 worker sonobouy check", func() {

	config, err := clientcmd.BuildConfigFromFlags("", tools.Kubeconfig)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
	kubeClient, err := kubernetes.NewForConfig(config)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
	localKubeConfigPath := "cluster1-sonobouy-config"

	defer ginkgo.GinkgoRecover()

	// do cluster installation within docker
	ginkgo.Context("[Test]when install a sonobouy cluster using docker", func() {
		clusterInstallYamlsPath := "e2e-install-cluster-sonobouy"
		kubeanNamespace := "kubean-system"
		kubeanClusterOpsName := "e2e-cluster1-install-sonobouy"

		// Create yaml for kuBean CR and related configuration
		installYamlPath := fmt.Sprint(tools.GetKuBeanPath(), clusterInstallYamlsPath)
		cmd := exec.Command("kubectl", "--kubeconfig="+tools.Kubeconfig, "apply", "-f", installYamlPath)
		out, _ := tools.DoCmd(*cmd)
		fmt.Println(out.String())

		// Check if the job and related pods have been created
		time.Sleep(30 * time.Second)
		pods, _ := kubeClient.CoreV1().Pods(kubeanNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=kubean-%s-job", kubeanClusterOpsName),
		})
		gomega.Expect(len(pods.Items)).NotTo(gomega.Equal(0))
		jobPodName := pods.Items[0].Name

		// Wait for job-related pod status to be succeeded
		for {
			pod, err := kubeClient.CoreV1().Pods(kubeanNamespace).Get(context.Background(), jobPodName, metav1.GetOptions{})
			ginkgo.GinkgoWriter.Printf("* wait for install job using docker related pod[%s] status: %s\n", pod.Name, pod.Status.Phase)
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed get job related pod")
			podStatus := string(pod.Status.Phase)
			if podStatus == "Succeeded" || podStatus == "Failed" {
				ginkgo.It("cluster deploy job related pod Status should be Succeeded", func() {
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

	time.Sleep(2 * time.Minute)
	// check kube-system pod status
	ginkgo.Context("[Test]When fetching kube-system pods status", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		podList, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to check kube-system pod status")
		for _, pod := range podList.Items {
			for {
				po, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), pod.Name, metav1.GetOptions{})
				ginkgo.GinkgoWriter.Printf("* wait for kube-system pod[%s] status: %s\n", po.Name, po.Status.Phase)
				podStatus := string(po.Status.Phase)
				if podStatus == "Running" || podStatus == "Failed" {
					ginkgo.It("every pod in kube-system should be in running status", func() {
						gomega.Expect(podStatus).To(gomega.Equal("Running"))
					})
					break
				}
				time.Sleep(1 * time.Minute)
			}
		}

		// check kube version before upgrade
		nodeList, _ := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		for _, node := range nodeList.Items {
			ginkgo.It("kube node version should be v1.22.12 before cluster upgrade", func() {
				gomega.Expect(node.Status.NodeInfo.KubeletVersion).To(gomega.Equal("v1.22.12"))
				gomega.Expect(node.Status.NodeInfo.KubeProxyVersion).To(gomega.Equal("v1.22.12"))
			})
		}

	})

	// sonobuoy run --sonobuoy-image docker.m.daocloud.io/sonobuoy/sonobuoy:v0.56.7 --plugin-env e2e.E2E_FOCUS=pods --plugin-env e2e.E2E_DRYRUN=true --wait
	ginkgo.Context("do sonobuoy checking ", func() {
		masterSSH := fmt.Sprintf("root@%s", tools.Vmipaddr)
		cmd := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "sonobuoy", "run", "--sonobuoy-image", "10.6.170.10:5000/sonobuoy/sonobuoy:v0.56.7", "--plugin-env", "e2e.E2E_FOCUS=pods", "--plugin-env", "e2e.E2E_DRYRUN=true", "--wait")
		out, _ := tools.DoCmd(*cmd)
		fmt.Println(out.String())

		sshcmd := exec.Command("sshpass", "-p", "root", "ssh", masterSSH, "sonobuoy", "status")
		sshout, _ := tools.DoCmd(*sshcmd)
		fmt.Println(sshout.String())

		ginkgo.GinkgoWriter.Printf("sonobuoy status result: %s\n", out.String())
		ginkgo.It("sonobuoy status checking result", func() {
			gomega.Expect(sshout.String()).Should(gomega.ContainSubstring("complete"))
			gomega.Expect(sshout.String()).Should(gomega.ContainSubstring("passed"))
		})
	})

	// check network configuration:
	// cat /proc/sys/net/ipv4/ip_forward: 1
	// cat /proc/sys/net/ipv4/tcp_tw_recycle: 0
	ginkgo.Context("do network configurations checking", func() {
		masterSSH := fmt.Sprintf("root@%s", tools.Vmipaddr)
		workerSSH := fmt.Sprintf("root@%s", tools.Vmipaddr2)
		masterCmd := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "cat", "/proc/sys/net/ipv4/ip_forward")
		workerCmd := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", workerSSH, "cat", "/proc/sys/net/ipv4/ip_forward")
		out1, _ := tools.DoCmd(*masterCmd)
		fmt.Println("out: ", out1.String())
		ginkgo.It("master net.ipv4.ip_forward result checking: ", func() {
			gomega.Expect(out1.String()).Should(gomega.ContainSubstring("1"))
		})
		out2, _ := tools.DoCmd(*workerCmd)
		fmt.Println("out: ", out2.String())
		ginkgo.It("worker net.ipv4.ip_forward result checking: ", func() {
			gomega.Expect(out2.String()).Should(gomega.ContainSubstring("1"))
		})

		masterCmd = exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "cat", "/proc/sys/net/ipv4/tcp_tw_recycle")
		workerCmd = exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", workerSSH, "cat", "/proc/sys/net/ipv4/tcp_tw_recycle")
		out3, _ := tools.DoCmd(*masterCmd)
		fmt.Println("out: ", out3.String())
		ginkgo.It("master net.ipv4.tcp_tw_recycle result checking: ", func() {
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("0"))
		})
		out4, _ := tools.DoCmd(*workerCmd)
		fmt.Println("out: ", out4.String())
		ginkgo.It("worker net.ipv4.tcp_tw_recycle result checking: ", func() {
			gomega.Expect(out4.String()).Should(gomega.ContainSubstring("0"))
		})
	})

	ginkgo.Context("[Test]check CNI: calico installed", func() {
		masterSSH := fmt.Sprintf("root@%s", tools.Vmipaddr)
		workerSSH := fmt.Sprintf("root@%s", tools.Vmipaddr2)
		// 1. check calico binary files in /opt/cni/bin
		masterCmd := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "ls", "/opt/cni/bin/")
		workerCmd := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", workerSSH, "ls", "/opt/cni/bin/")
		out1, _ := tools.DoCmd(*masterCmd)
		fmt.Println("out1: ", out1.String())
		ginkgo.It("master calico checking: ", func() {
			gomega.Expect(out1.String()).Should(gomega.ContainSubstring("calico"))
		})
		out2, _ := tools.DoCmd(*workerCmd)
		fmt.Println("out2: ", out2.String())
		ginkgo.It("worker calico checking: ", func() {
			gomega.Expect(out2.String()).Should(gomega.ContainSubstring("calico"))
		})

		// 2. check calicoctl
		masterCmd = exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "calicoctl", "version")
		out3, _ := tools.DoCmd(*masterCmd)
		fmt.Println("out1: ", out3.String())
		ginkgo.It("master calicoctl checking: ", func() {
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("Client Version"))
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("Cluster Version"))
			gomega.Expect(out3.String()).Should(gomega.ContainSubstring("kubespray,kubeadm,kdd"))
		})

		// 3. create 2 new pods, pod can ping pod, node ping pod
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		nginx1 := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx1",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx",
						Image: "nginx:alpine",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 80,
							},
						}},
				},
			},
		}
		nginx2 := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx2",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx",
						Image: "nginx:alpine",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 88,
							},
						}},
				},
			},
		}
		_, err = kubeClient.CoreV1().Pods("kube-system").Create(context.Background(), nginx1, metav1.CreateOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to create nginxPod1")
		time.Sleep(1 * time.Minute)
		_, err = kubeClient.CoreV1().Pods("kube-system").Create(context.Background(), nginx2, metav1.CreateOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to create nginxPod2")
		time.Sleep(1 * time.Minute)

		pod1, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), "nginx1", metav1.GetOptions{})
		nginx1Ip := string(pod1.Status.PodIP)
		ginkgo.It("nginxPod1 should be in running status", func() {
			gomega.Expect(string(pod1.Status.Phase)).To(gomega.Equal("Running"))
		})
		pod2, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), "nginx2", metav1.GetOptions{})
		nginx2Ip := string(pod2.Status.PodIP)
		ginkgo.It("nginxPod1 should be in running status", func() {
			gomega.Expect(string(pod2.Status.Phase)).To(gomega.Equal("Running"))
		})
		// 3.1 node ping 2 pods
		pingNginx1IpCmd1 := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "ping", "-c 1", nginx1Ip)
		pingNginx1IpCmd1Out, _ := tools.DoCmd(*pingNginx1IpCmd1)
		fmt.Println("node ping nginx pod 1: ", pingNginx1IpCmd1Out.String())
		ginkgo.It("node ping nginx pod 1 succuss: ", func() {
			gomega.Expect(pingNginx1IpCmd1Out.String()).Should(gomega.ContainSubstring("1 received"))
		})
		pingNginx2IpCmd1 := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", masterSSH, "ping", "-c 2", nginx2Ip)
		pingNgin21IpCmd1Out, _ := tools.DoCmd(*pingNginx2IpCmd1)
		fmt.Println("node ping nginx pod 2: ", pingNgin21IpCmd1Out.String())
		ginkgo.It("node ping nginx pod 2 succuss: ", func() {
			gomega.Expect(pingNgin21IpCmd1Out.String()).Should(gomega.ContainSubstring("1 received"))
		})
		// 3.2 pod ping pod
		podsPingCmd1 := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no", masterSSH, "kubectl", "exec", "-it", "nginx1",
			"--", "ping", "-w 2", "-c 1", nginx2Ip)
		podsPingCmdOut1, _ := tools.DoCmd(*podsPingCmd1)
		fmt.Println("pod ping pod: ", podsPingCmdOut1.String())
		ginkgo.It("pod ping pod succuss: ", func() {
			gomega.Expect(podsPingCmdOut1.String()).Should(gomega.ContainSubstring("1 received"))
		})
		podsPingCmd2 := exec.Command("sshpass", "-p", "root", "ssh", "-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no", masterSSH, "kubectl", "exec", "-it", "nginx2",
			"--", "ping", "-w 2", "-c 1", nginx1Ip)
		podsPingCmdOut2, _ := tools.DoCmd(*podsPingCmd2)
		fmt.Println("pod ping pod: ", podsPingCmdOut2.String())
		ginkgo.It("pod ping pod succuss: ", func() {
			gomega.Expect(podsPingCmdOut2.String()).Should(gomega.ContainSubstring("1 received"))
		})
	})

	// cluster upgrade from v1.22.12 to v1.23.7
	ginkgo.Context("do cluster upgrade from v1.22.12 to v1.23.7", func() {
		clusterInstallYamlsPath := "e2e-upgrade-cluster"
		kubeanNamespace := "kubean-system"
		kubeanClusterOpsName := "e2e-upgrade-cluster"

		// Create yaml for kuBean CR and related configuration
		installYamlPath := fmt.Sprint(tools.GetKuBeanPath(), clusterInstallYamlsPath)
		cmd := exec.Command("kubectl", "--kubeconfig="+tools.Kubeconfig, "apply", "-f", installYamlPath)
		out, _ := tools.DoCmd(*cmd)
		fmt.Println(out.String())

		// Check if the job and related pods have been created
		time.Sleep(30 * time.Second)
		config, _ = clientcmd.BuildConfigFromFlags("", tools.Kubeconfig)
		kubeClient, _ = kubernetes.NewForConfig(config)
		pods, _ := kubeClient.CoreV1().Pods(kubeanNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=kubean-%s-job", kubeanClusterOpsName),
		})
		gomega.Expect(len(pods.Items)).NotTo(gomega.Equal(0))
		jobPodName := pods.Items[0].Name

		// Wait for job-related pod status to be succeeded
		for {
			pod, err := kubeClient.CoreV1().Pods(kubeanNamespace).Get(context.Background(), jobPodName, metav1.GetOptions{})
			ginkgo.GinkgoWriter.Printf("* wait for upgrade job related pod[%s] status: %s\n", pod.Name, pod.Status.Phase)
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed get job related pod")
			podStatus := string(pod.Status.Phase)
			if podStatus == "Succeeded" || podStatus == "Failed" {
				ginkgo.It("cluster upgrade job related pod Status should be Succeeded", func() {
					gomega.Expect(podStatus).To(gomega.Equal("Succeeded"))
				})
				break
			}
			time.Sleep(1 * time.Minute)
		}
	})

	time.Sleep(1 * time.Minute)
	// check kube pods status after upgrade from v1.22.12 to v1.23.7
	ginkgo.Context("When fetching kube-system pods status after upgrade from v1.22.12 to v1.23.7", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		podList, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to check kube-system pod status")
		for _, pod := range podList.Items {
			for {
				po, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), pod.Name, metav1.GetOptions{})
				ginkgo.GinkgoWriter.Printf("* wait for upgrade job from v1.22.12 to v1.23.7 related pod[%s] status: %s\n", po.Name, po.Status.Phase)
				podStatus := string(po.Status.Phase)
				if podStatus == "Running" || podStatus == "Failed" {
					ginkgo.It("every pod in kube-system after upgrade from v1.22.12 to v1.23.7 should be in running status", func() {
						gomega.Expect(podStatus).To(gomega.Equal("Running"))
					})
					break
				}
				time.Sleep(1 * time.Minute)
			}
		}
	})
	// check kube version after upgrade from v1.22.12 to v1.23.7
	ginkgo.Context("check kube version after upgrade from v1.22.12 to v1.23.7", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		nodeList, _ := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		for _, node := range nodeList.Items {
			ginkgo.It("kube node version should be v1.23.7", func() {
				gomega.Expect(node.Status.NodeInfo.KubeletVersion).To(gomega.Equal("v1.23.7"))
				gomega.Expect(node.Status.NodeInfo.KubeProxyVersion).To(gomega.Equal("v1.23.7"))
			})
		}
	})

	// cluster upgrade from v1.23.7 to v1.24.3
	ginkgo.Context("do cluster upgrade from from v1.23.7 to v1.24.3", func() {
		clusterInstallYamlsPath := "e2e-upgrade-cluster24"
		kubeanNamespace := "kubean-system"
		kubeanClusterOpsName := "e2e-upgrade-cluster24"

		// Create yaml for kuBean CR and related configuration
		installYamlPath := fmt.Sprint(tools.GetKuBeanPath(), clusterInstallYamlsPath)
		cmd := exec.Command("kubectl", "--kubeconfig="+tools.Kubeconfig, "apply", "-f", installYamlPath)
		out, _ := tools.DoCmd(*cmd)
		fmt.Println(out.String())

		// Check if the job and related pods have been created
		time.Sleep(30 * time.Second)
		config, _ = clientcmd.BuildConfigFromFlags("", tools.Kubeconfig)
		kubeClient, _ = kubernetes.NewForConfig(config)
		pods, _ := kubeClient.CoreV1().Pods(kubeanNamespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=kubean-%s-job", kubeanClusterOpsName),
		})
		gomega.Expect(len(pods.Items)).NotTo(gomega.Equal(0))
		jobPodName := pods.Items[0].Name

		// Wait for job-related pod status to be succeeded
		for {
			pod, err := kubeClient.CoreV1().Pods(kubeanNamespace).Get(context.Background(), jobPodName, metav1.GetOptions{})
			ginkgo.GinkgoWriter.Printf("* wait for upgrade job from v1.23.7 to v1.24.3 related pod[%s] status: %s\n", pod.Name, pod.Status.Phase)
			gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed get job related pod")
			podStatus := string(pod.Status.Phase)
			if podStatus == "Succeeded" || podStatus == "Failed" {
				ginkgo.It("cluster upgrade from v1.23.7 to v1.24.3 job related pod Status should be Succeeded", func() {
					gomega.Expect(podStatus).To(gomega.Equal("Succeeded"))
				})
				break
			}
			time.Sleep(1 * time.Minute)
		}
	})

	time.Sleep(1 * time.Minute)
	// check kube pods status after upgrade from v1.23.7 to v1.24.3
	ginkgo.Context("When fetching kube-system pods status after upgrade from v1.23.7 to v1.24.3", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		podList, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to check kube-system pod status")
		for _, pod := range podList.Items {
			for {
				po, _ := kubeClient.CoreV1().Pods("kube-system").Get(context.Background(), pod.Name, metav1.GetOptions{})
				ginkgo.GinkgoWriter.Printf("* wait for upgrade job from v1.23.7 to v1.24.3 related pod[%s] status: %s\n", po.Name, po.Status.Phase)
				podStatus := string(po.Status.Phase)
				if podStatus == "Running" || podStatus == "Failed" {
					ginkgo.It("every pod in kube-system after upgrade from v1.23.7 to v1.24.3 should be in running status", func() {
						gomega.Expect(podStatus).To(gomega.Equal("Running"))
					})
					break
				}
				time.Sleep(1 * time.Minute)
			}
		}
	})
	// check kube version after upgrade from v1.23.7 to v1.24.3
	ginkgo.Context("check kube version after upgrade from v1.23.7 to v1.24.3", func() {
		config, err = clientcmd.BuildConfigFromFlags("", localKubeConfigPath)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed build config")
		kubeClient, err = kubernetes.NewForConfig(config)
		gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed new client set")
		nodeList, _ := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		for _, node := range nodeList.Items {
			ginkgo.It("kube node version should be v1.24.3", func() {
				gomega.Expect(node.Status.NodeInfo.KubeletVersion).To(gomega.Equal("v1.24.3"))
				gomega.Expect(node.Status.NodeInfo.KubeProxyVersion).To(gomega.Equal("v1.24.3"))
			})
		}
	})

})

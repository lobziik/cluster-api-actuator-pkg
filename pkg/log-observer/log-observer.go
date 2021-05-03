package log_observer

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"k8s.io/client-go/kubernetes"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[LogObserver] Observer should", func() {
	var client *kubernetes.Clientset
	var rtclient runtimeclient.Client
	var logsObserver *framework.PodLogsObserver

	defer GinkgoRecover()

	BeforeEach(func() {
		var err error

		fmt.Println("Before")
		client, err = framework.LoadClientset()
		Expect(err).NotTo(HaveOccurred())
		rtclient, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		logsObserver = framework.NewPodLogsObserver(client, ContainSubstring("SIGQUIT"))

	})

	AfterEach(func() {

	})

	It("dummy", func() {
		selector := map[string]string{
			"api":     "clusterapi",
			"k8s-app": "controller",
		}
		pods, err := framework.GetPods(rtclient, selector)
		Expect(err).NotTo(HaveOccurred())
		err, resultCh := logsObserver.Start(context.Background(), pods)
		if err != nil {
			panic(err)
		}
		for {
			captured := <-resultCh
			fmt.Println(captured.Message())
		}
	})
})

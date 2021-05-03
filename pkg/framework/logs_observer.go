package framework

import (
	"bufio"
	"container/list"
	"container/ring"
	"context"
	"fmt"
	gomegatype "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

type PodLogsObserver struct {
	startTime   time.Time
	results     chan CapturedLogsBatch
	client      *kubernetes.Clientset
	matcher     gomegatype.GomegaMatcher
	linesBefore int
	linesAfter  int
}

type CapturedLogsBatch struct {
	PodName      string
	PodContainer string
	cause        string
	before       *list.List
	after        *list.List
}

func (lb CapturedLogsBatch) Message() string {
	var msgAccumulator string
	msgAccumulator = ""
	fmt.Println(lb.before.Len())
	for e := lb.before.Front(); e != nil; e = e.Next() {

		msgAccumulator = msgAccumulator + e.Value.(string) + "\n"
	}
	msgAccumulator += lb.cause + "\n"
	for e := lb.after.Front(); e != nil; e = e.Next() {
		msgAccumulator = msgAccumulator + e.Value.(string) + "\n"
	}
	return msgAccumulator
}

func NewPodLogsObserver(client *kubernetes.Clientset, matchers gomegatype.GomegaMatcher) *PodLogsObserver {
	return &PodLogsObserver{
		startTime: time.Now(),
		client:    client,
		matcher:   matchers,
	}
}

func (lw *PodLogsObserver) setupPodLogsObserver(ctx context.Context, pod corev1.Pod) error {
	tailLines := int64(100)
	for _, container := range pod.Status.ContainerStatuses {
		podLogOpts := corev1.PodLogOptions{
			Container: container.Name,
			TailLines: &tailLines,
			Follow:    true,
		}

		go func(
			ctx context.Context,
			namespace string,
			podName string,
			containerName string,
			logOpts corev1.PodLogOptions,
			resultsCh chan<- CapturedLogsBatch,
			pod corev1.Pod,
		) {

			capturedQueue := list.New()
			captureBeforeRingBuf := ring.New(lw.linesBefore)
			//captureAfterRingBuf := ring.New(lw.linesAfter)

			req := lw.client.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
			podLogs, err := req.Stream(ctx)
			if err != nil {
				panic(err)
			}
			defer podLogs.Close()

			fmt.Println(podName, "    ", containerName)
			scanner := bufio.NewScanner(podLogs)
			for scanner.Scan() {
				logLine := scanner.Text()
				captureBeforeRingBuf.Value = logLine
				captureBeforeRingBuf = captureBeforeRingBuf.Next()
				match, err := lw.matcher.Match(logLine)
				if err != nil {
					panic(err)
				}
				if match {
					linesBeforeList := list.New()
					captureBeforeRingBuf.Do(func(e interface{}) {
						linesBeforeList.PushBack(e)
					})
					capturedQueue.PushBack(
						CapturedLogsBatch{
							podName,
							containerName,
							logLine,
							linesBeforeList,
							list.New(),
						},
					)
				}

				if capturedQueue.Len() > 0 {
					for e := capturedQueue.Front(); e != nil; e = e.Next() {
						lb := e.Value.(CapturedLogsBatch)
						if lw.linesAfter > lb.after.Len() {
							lb.after.PushBack(logLine)
						} else {
							resultsCh <- lb
							capturedQueue.Remove(e)
						}
					}
				}

			}

			fmt.Println("queue len:", capturedQueue.Len())
			if scanner.Err() != nil {
				fmt.Println("EOF, pod seems dead")
			} else {
				fmt.Println(scanner.Err())
			}

		}(ctx, pod.Namespace, pod.Name, container.Name, podLogOpts, lw.results, pod)
	}
	return nil
}

func (lw *PodLogsObserver) Start(ctx context.Context, pods *corev1.PodList) (error, <-chan CapturedLogsBatch) {
	if lw.matcher == nil {
		return fmt.Errorf("Can not start without matchers specified"), nil
	}
	lw.startTime = time.Now()
	lw.linesBefore = 8
	lw.linesAfter = 64
	lw.results = make(chan CapturedLogsBatch)
	for _, pod := range pods.Items {
		err := lw.setupPodLogsObserver(ctx, pod)
		if err != nil {
			panic(err)
		}
	}
	return nil, lw.results
}

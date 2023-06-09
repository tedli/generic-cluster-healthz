package healthz

import (
	"context"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tedli/generic-cluster-healthz/pkg/utils"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	lastHealthTimestamp time.Time
)

const (
	readyType = "Ready"
)

func nodeGetConditions(node corev1.Node) []corev1.NodeCondition {
	return node.Status.Conditions
}

func nodeGetStatus(_ corev1.Node, condition corev1.NodeCondition) (string, bool) {
	return string(condition.Type), condition.Status == corev1.ConditionTrue
}

func isReady[Object, Condition any](object Object, getConditions func(Object) []Condition,
	getStatus func(Object, Condition) (string, bool)) (ready bool) {

	if conditions := getConditions(object); conditions == nil {
		return
	} else {
		for _, condition := range conditions {
			if typ, is := getStatus(object, condition); typ == readyType && is {
				ready = true
				break
			}
		}
	}
	return
}

func checkResource[List, Object, Condition any](list List, n int,
	getItems func(List) []Object,
	getConditions func(Object) []Condition,
	getStatus func(Object, Condition) (string, bool)) bool {

	var ready int
	if items := getItems(list); items != nil {
		for _, item := range items {
			if isReady(item, getConditions, getStatus) {
				ready++
			}
			if ready >= n {
				return true
			}
		}
	}
	return false
}

func getNodes(list *corev1.NodeList) []corev1.Node {
	return list.Items
}

func getPods(list *corev1.PodList) []corev1.Pod {
	return list.Items
}

func getPodConditions(pod corev1.Pod) []corev1.PodCondition {
	return pod.Status.Conditions
}

func getPodStatus(pod corev1.Pod, condition corev1.PodCondition) (string, bool) {
	return string(condition.Type), !pod.Spec.HostNetwork && condition.Status == corev1.ConditionTrue
}

func check(ctx context.Context, logger logr.Logger,
	minNode, minPod int, client kubernetes.Interface) (health bool, err error) {

	if minNode > 0 {
		var nl *corev1.NodeList
		if nl, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
			logger.Error(err, "list node failed")
			return
		}
		if health = checkResource(nl, minNode, getNodes, nodeGetConditions, nodeGetStatus); !health {
			return
		}
	}
	if minPod > 0 {
		var pl *corev1.PodList
		if pl, err = client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "status.phase=Running"}); err != nil {
			logger.Error(err, "list pod failed")
			return
		}
		if health = checkResource(pl, minPod, getPods, getPodConditions, getPodStatus); !health {
			return
		}
	}
	return
}

var (
	debugStatus *bool
)

func newService(rootLogger logr.Logger, client kubernetes.Interface, minNode, minPod int, checkDuration time.Duration) (mux *router.Router) {
	logger := rootLogger.WithName("healthz")
	mux = router.New()
	mux.GET("/healthz", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		_, _ = ctx.WriteString("ok")
	})
	mux.GET("/readyz", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		_, _ = ctx.WriteString("ok")
	})
	mux.GET("/metrics", fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler()))
	mux.PUT("/debug/cluster-healthiness", func(ctx *fasthttp.RequestCtx) {
		healthy := ctx.QueryArgs().GetBool("healthy")
		debugStatus = utils.Pointer(healthy)
		logger.Info("set debugging status", "healthy", healthy)
		ctx.SetStatusCode(fasthttp.StatusOK)
		_, _ = ctx.WriteString(strconv.FormatBool(healthy))
		return
	})
	mux.DELETE("/debug/cluster-healthiness", func(ctx *fasthttp.RequestCtx) {
		debugStatus = nil
		logger.Info("clear debugging status")
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return
	})
	mux.GET("/cluster-healthz", func(ctx *fasthttp.RequestCtx) {
		if debugStatus != nil {
			logger.Info("returning debugging status", "status", *debugStatus)
			if health := *debugStatus; health {
				ctx.SetStatusCode(fasthttp.StatusOK)
				_, _ = ctx.WriteString("ok")
			} else {
				ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
				_, _ = ctx.WriteString("bad")
			}
			return
		}
		now := time.Now()
		if checkDuration >= 0 && !lastHealthTimestamp.IsZero() && now.Sub(lastHealthTimestamp) < checkDuration {
			ctx.SetStatusCode(fasthttp.StatusOK)
			_, _ = ctx.WriteString("ok")
			return
		}
		if health, err := check(ctx, logger, minNode, minPod, client); err != nil {
			logger.Error(err, "check cluster health failed")
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			_, _ = ctx.WriteString(err.Error())
			return
		} else {
			logger.Info("check cluster health succeed", "health", health)
			if !health {
				ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
				_, _ = ctx.WriteString("bad")
				return
			} else {
				ctx.SetStatusCode(fasthttp.StatusOK)
				_, _ = ctx.WriteString("ok")
				lastHealthTimestamp = now
				return
			}
		}
	})
	return
}

type fastHTTPLogger struct {
	logger logr.Logger
}

func (fhl fastHTTPLogger) Printf(format string, args ...interface{}) {
	fhl.logger.Info(fmt.Sprintf(format, args...))
}

func newServer(bindAddress string, logger logr.Logger, httpLogger fasthttp.Logger,
	handler fasthttp.RequestHandler, certPath, keyPath string) func(context.Context) error {

	return func(ctx context.Context) (err error) {
		https := certPath != "" && keyPath != ""
		logger := logger.WithValues("https", https, "bind", bindAddress)
		var ln net.Listener
		if ln, err = net.Listen("tcp4", bindAddress); err != nil {
			logger.Error(err, "listen http port failed")
			return
		} else {
			server := &fasthttp.Server{
				IdleTimeout:        30 * time.Second,
				TCPKeepalive:       true,
				TCPKeepalivePeriod: 30 * time.Second,
				MaxConnsPerIP:      200,
				Handler:            handler,
				Logger:             httpLogger,
			}
			defer logger.Info("serving finished")
			go func() {
				<-ctx.Done()
				if e := server.Shutdown(); e != nil {
					logger.Error(e, "shutdown server failed")
				}
			}()
			if https {
				err = server.ServeTLS(ln, certPath, keyPath)
			} else {
				err = server.Serve(ln)
			}
		}
		return
	}
}

func NewHealthz(bindAddress, httpsBindAddress string, rootLogger logr.Logger,
	client kubernetes.Interface, minNode, minPod int, certPath, keyPath string,
	checkDuration time.Duration) func(context.Context) error {
	logger := rootLogger.WithName("httpServer")
	httpLogger := &fastHTTPLogger{logger: logger.WithName("fastHttp")}
	mux := newService(rootLogger, client, minNode, minPod, checkDuration)
	handler := mux.Handler
	return func(ctx context.Context) (err error) {
		https := httpsBindAddress != "" && certPath != "" && keyPath != ""
		servers := make([]func(context.Context) error, 0, 2)
		servers = append(servers, newServer(bindAddress, logger, httpLogger,
			handler, "", ""))
		if https {
			servers = append(servers, newServer(httpsBindAddress, logger, httpLogger,
				handler, certPath, keyPath))
		}
		join := new(sync.WaitGroup)
		join.Add(len(servers))
		for _, server := range servers {
			go func(serve func(context.Context) error) {
				defer join.Done()
				if e := serve(ctx); e != nil {
					logger.Error(e, "serve failed")
					err = multierr.Append(err, e)
				}
			}(server)
		}
		join.Wait()
		return
	}
}

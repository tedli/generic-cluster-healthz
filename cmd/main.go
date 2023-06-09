package main

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tedli/generic-cluster-healthz/pkg/healthz"
	"github.com/tedli/generic-cluster-healthz/pkg/logging"
	"github.com/tedli/generic-cluster-healthz/pkg/utils"
	"github.com/tedli/generic-cluster-healthz/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	clientQPS              = "client-qps"
	clientBurst            = "client-burst"
	httpBindAddress        = "http-bind-address"
	httpsBindAddress       = "https-bind-address"
	minNonHostNetworkedPod = "min-pod"
	minNode                = "min-node"
	checkDuration          = "check-duration"
	certPath               = "cert-path"
	keyPath                = "key-path"
)

func newRootCommand() (cmd *cobra.Command, _ error) {
	var initViper func() error
	cmd = &cobra.Command{
		Use:   "healthz",
		Short: "A generic cluster healthiness checker.",
		Long:  "A generic cluster healthiness checker, which checks nodes ready and pods.",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(_ *cobra.Command, _ []string) (err error) {
			rootLogger := logging.NewLogger()
			logger := rootLogger.WithName("cmd")
			defer logger.Info("quit")
			if initViper != nil {
				if err = initViper(); err != nil {
					logger.Error(err, "init viper failed")
					return
				}
			}
			var cfg *rest.Config
			if cfg, err = ctrl.GetConfig(); err != nil {
				logger.Error(err, "get manager config failed")
				return
			}
			cfg.UserAgent = fmt.Sprintf("%s/%s", version.Name, version.Version)
			qps := viper.GetFloat64(clientQPS)
			cfg.QPS = float32(qps)
			cfg.Burst = viper.GetInt(clientBurst)
			var client kubernetes.Interface
			if client, err = kubernetes.NewForConfig(cfg); err != nil {
				logger.Error(err, "create kubernetes client failed")
				return
			}
			bind := viper.GetString(httpBindAddress)
			https := viper.GetString(httpsBindAddress)
			nodes := viper.GetInt(minNode)
			pods := viper.GetInt(minNonHostNetworkedPod)
			duration := viper.GetDuration(checkDuration)
			cert := viper.GetString(certPath)
			key := viper.GetString(keyPath)
			logger.Info("creating checker", "nodes", nodes, "pods", pods, "duration", duration)
			checker := healthz.NewHealthz(bind, https, rootLogger, client, nodes, pods, cert, key, duration)
			ctx := ctrl.SetupSignalHandler()
			logger.Info("checker started",
				"builtAt", version.BuiltAt, "commitId", version.CommitID)
			if err = checker(ctx); err != nil {
				logger.Error(err, "run checker failed")
			}
			return
		},
	}
	fs := flag.NewFlagSet("", flag.PanicOnError)
	pfs := cmd.PersistentFlags()
	logging.Init(fs, pfs)
	cmd.Flags().AddGoFlagSet(fs)
	pfs.Float32(clientQPS, 5.0, "The maximum QPS to the master from this client.")
	utils.BindEnv(clientQPS)
	pfs.Int(clientBurst, 10, "The maximum burst for throttle.")
	utils.BindEnv(clientBurst)
	pfs.String(httpBindAddress, "0.0.0.0:8080", "HTTP binding address.")
	utils.BindEnv(httpBindAddress)
	pfs.String(httpsBindAddress, "0.0.0.0:8443", "HTTPS binding address.")
	utils.BindEnv(httpsBindAddress)
	pfs.Int(minNonHostNetworkedPod, 1, "Minimal ready non host network pods.")
	utils.BindEnv(minNonHostNetworkedPod)
	pfs.Int(minNode, 1, "Minimal ready nodes.")
	utils.BindEnv(minNode)
	pfs.String(certPath, "", "The tls cert pem file full path.")
	utils.BindEnv(certPath)
	pfs.String(keyPath, "", "The tls private key pem file full path.")
	utils.BindEnv(keyPath)
	pfs.Duration(checkDuration, 10*time.Minute, "Interval of do check since last health check.")
	utils.BindEnv(checkDuration)
	initViper = utils.RegisterToViper(pfs, "healthz")
	return
}

func main() {
	if app, err := newRootCommand(); err != nil {
		panic(err)
	} else if err = app.Execute(); err != nil {
		panic(err)
	}
}

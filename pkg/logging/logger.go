package logging

import (
	"flag"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

func Init(fs *flag.FlagSet, pfs *pflag.FlagSet) {
	klog.InitFlags(fs)
	verbosity := fs.Lookup("v")
	pfs.AddFlag(pflag.PFlagFromGoFlag(&flag.Flag{
		Name:     "vklog",
		Value:    verbosity.Value,
		DefValue: verbosity.DefValue,
		Usage:    verbosity.Usage + ". Like -v flag. ex: --vklog=9.",
	}))
}

func NewLogger() (lgr logr.Logger) {
	return klog.NewKlogr()
}

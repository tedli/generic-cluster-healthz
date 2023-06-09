package utils

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
)

func RegisterToViper(pfs *pflag.FlagSet, configName string) func() error {
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/karmada")
	viper.AddConfigPath(".")
	viper.AddConfigPath("debug")
	return func() (err error) {
		if err = viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return
			}
			err = nil
		} else {
			viper.WatchConfig()
		}
		err = viper.BindPFlags(pfs)
		return
	}
}

const (
	karmadaEnvPrefix = "KARMADA_"
)

func getEnvName(option string) string {
	under := strings.Replace(option, "-", "_", -1)
	upper := strings.ToUpper(under)
	return karmadaEnvPrefix + upper
}

func BindEnv(key string) {
	viper.MustBindEnv(key, getEnvName(key))
}

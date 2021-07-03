package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"github.wdf.sap.corp/i350641/edgify-app-server/internal"
	log "k8s.io/klog"
)

func main() {
	// set configurations
	log.V(2).Info("loading configuration")
	cfg := viper.New()
	cfgFile := os.Getenv("CONFIG_FILE")
	if cfgFile != "" {
		cfg.SetConfigFile(cfgFile)
		if err := cfg.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}
	cfg.SetDefault(internal.CfgExecPostTimeOut, internal.DefExecPostTimeOut)
	cfg.SetDefault(internal.CfgPostTimeOut, internal.DefPostTimeOut)
	cfg.SetDefault(internal.CfgPort, internal.DefPort)

	// init Exec Client
	stopCh := make(chan struct{})
	defer close(stopCh)
	execClient := internal.NewExecClient(stopCh)
	execClient.Run()

	// init App Service
	appSrv := internal.NewAppService(cfg, execClient)
	if err := appSrv.Run(fmt.Sprintf(":%d", cfg.GetUint(internal.CfgPort))); err != nil {
		log.Fatalf("failed to start service. %s", err.Error())
	}
	log.V(2).Infof("stopping...")
}
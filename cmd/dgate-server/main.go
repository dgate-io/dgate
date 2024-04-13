package main

import (
	"fmt"
	"os"

	"runtime/debug"

	"github.com/dgate-io/dgate/internal/admin"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/util"
	"github.com/spf13/pflag"
)

var version string = "dev"

func main() {
	showVersion := pflag.BoolP("version", "v", false, "print current version")
	configPath := pflag.StringP("config", "c", "", "path to config file")
	help := pflag.BoolP("help", "h", false, "show help")

	pflag.Parse()

	if *help {
		pflag.Usage()
		return
	}

	// get version from build info when installed using `go install``
	buildInfo, ok := debug.ReadBuildInfo()
	bversion := buildInfo.Main.Version
	if ok && bversion != "" && bversion != "(devel)" {
		version = buildInfo.Main.Version
	}

	if version == "dev" {
		version = fmt.Sprintf("dev/PID:%d", os.Getpid())
	}

	if *showVersion {
		println(version)
	} else {
		if !util.EnvVarCheckBool("DG_DISABLE_BANNER") {
			fmt.Println(
				"_________________      _____      \n" +
					"___  __ \\_  ____/_____ __  /_____ \n" +
					"__  / / /  / __ _  __ `/  __/  _ \\\n" +
					"_  /_/ // /_/ / / /_/ // /_ /  __/\n" +
					"/_____/ \\____/  \\__,_/ \\__/ \\___/ \n" +
					"                                   \n" +
					"DGate - API Gateway Server (" + version + ")\n" +
					"-----------------------------------\n",
			)
		}
		dgateConfig, err := config.LoadConfig(*configPath)
		if err != nil {
			panic(err)
		}

		proxyState, err := proxy.StartProxyGateway(dgateConfig)
		if err != nil {
			panic(err)
		}

		admin.StartAdminAPI(dgateConfig, proxyState)
	}
}

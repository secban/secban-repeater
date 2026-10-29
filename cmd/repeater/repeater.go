package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/profiler"
	"gitlab.com/secban/secban-agent/version"
	"gitlab.com/secban/secban-repeater/config"
	"gitlab.com/secban/secban-repeater/repeater"
)

func main() {
	version.CLI()

	conf := loadConf()

	zerolog.SetGlobalLevel(conf.LogLevel)

	rep := repeater.New(conf.Agents, conf.Timeout, conf.RulesMap)

	ctx := context.Background()

	if conf.Profiler != "" {
		go profiler.Listen(ctx, conf.Profiler)
	}

	log.Info().
		Str("socket", conf.SocketPath).
		Int("agents", len(conf.Agents)).
		Msg("secban-repeater starting")

	listenErr := ipc.ListenUnixSocket(conf.SocketPath, func(msg ipc.Message) error {
		rep.Repeat(ctx, msg)

		return nil
	})
	if listenErr != nil {
		log.Fatal().Err(listenErr).Msg("failed to listen on unix socket")
	}
}

func loadConf() *config.Config {
	var configDir string
	//nolint:revive,nolintlint // Not a real deep-exit, extracted from main for readability.
	flag.StringVar(&configDir, "config", "/etc/secban", "path to config directory")
	//nolint:revive,nolintlint // Not a real deep-exit, extracted from main for readability.
	flag.Parse()

	conf, confErr := config.LoadConfig(configDir)
	if confErr != nil {
		log.Fatal().Err(confErr).Msg("failed to load config")
	}

	return conf
}

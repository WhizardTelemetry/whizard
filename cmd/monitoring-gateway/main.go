/*
Copyright 2024 the Whizard Authors.

Licensed under Apache License, Version 2.0 with a few additional conditions.

You may obtain a copy of the License at

    https://github.com/WhizardTelemetry/whizard/blob/main/LICENSE
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"
	"go.uber.org/automaxprocs/maxprocs"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/logging"
)

func main() {
	// We use mmaped resources in most of the components so hardcode PanicOnFault to true. This allows us to recover (if we can e.g if queries
	// are temporarily accessing unmapped memory).
	debug.SetPanicOnFault(true)

	if os.Getenv("DEBUG") != "" {
		runtime.SetMutexProfileFraction(10)
		runtime.SetBlockProfileRate(10)
	}

	app := extkingpin.NewApp(kingpin.New(filepath.Base(os.Args[0]), "Whizard is a Prometheus Long-Term Storage powered by Thanos").Version(version.Print("Whizard")))
	debugName := app.Flag("debug.name", "Name to add as prefix to log lines.").Hidden().String()
	logLevel := app.Flag("log.level", "Log filtering level.").
		Default("info").Enum("error", "warn", "info", "debug")
	logFormat := app.Flag("log.format", "Log format to use. Possible options: logfmt or json.").
		Default(logging.LogFormatLogfmt).Enum(logging.LogFormatLogfmt, logging.LogFormatJSON)

	registerGateway(app)
	registerAgentProxy(app)

	cmd, setup := app.Parse()
	logger := logging.NewLogger(*logLevel, *logFormat, *debugName)

	// Running in container with limits but with empty/wrong value of GOMAXPROCS env var could lead to throttling by cpu
	// maxprocs will automate adjustment by using cgroups info about cpu limit if it set as value for runtime.GOMAXPROCS.
	undo, err := maxprocs.Set(maxprocs.Logger(func(template string, args ...interface{}) {
		level.Debug(logger).Log("msg", fmt.Sprintf(template, args))
	}))
	defer undo()
	if err != nil {
		level.Warn(logger).Log("warn", errors.Wrapf(err, "failed to set GOMAXPROCS: %v", err))
	}

	metrics := prometheus.NewRegistry()
	metrics.MustRegister(
		versioncollector.NewCollector("whizard"),
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")}),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Some packages still use default Register. Replace to have those metrics.
	prometheus.DefaultRegisterer = metrics

	var g run.Group
	var tracer opentracing.Tracer

	// Create a signal channel to dispatch reload events to sub-commands.
	reloadCh := make(chan struct{}, 1)

	if err := setup(&g, logger, metrics, tracer, reloadCh, *logLevel == "debug"); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		level.Error(logger).Log("err", fmt.Sprintf("%+v", errors.Wrapf(err, "preparing %s command failed", cmd)))
		os.Exit(1)
	}

	// Listen for termination signals.
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			close(cancel)
		})
	}

	// Listen for reload signals.
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return reload(logger, cancel, reloadCh)
		}, func(error) {
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		level.Error(logger).Log("err", fmt.Sprintf("%+v", errors.Wrapf(err, "%s command failed", cmd)))
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "exiting")
}

func interrupt(logger log.Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		level.Info(logger).Log("msg", "caught signal. Exiting.", "signal", s)
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}

func reload(logger log.Logger, cancel <-chan struct{}, r chan<- struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		select {
		case s := <-c:
			level.Info(logger).Log("msg", "caught signal. Reloading.", "signal", s)
			select {
			case r <- struct{}{}:
				level.Info(logger).Log("msg", "reload dispatched.")
			default:
			}
		case <-cancel:
			return errors.New("canceled")
		}
	}
}

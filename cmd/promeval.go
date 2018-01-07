package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ghodss/yaml"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/config"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/pkg/api/v1"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Playground for Prometheus scrape configuration.")
	app.Version(version.Print("promeval"))
	app.HelpFlag.Short('h')

	targetsCmd := app.Command("targets", "Evaluate and prints all jobs' scrape targets from given configuration file.")
	configFile := targetsCmd.Arg(
		"config-file",
		"The Prometheus config file to evaluate.",
	).Required().ExistingFile()
	configMapItem := targetsCmd.Flag(
		"configmap-item",
		"If specified, config-file will be treated as configmap with Prometheus config as item under this key.",
	).String()
	format := targetsCmd.Flag("output", "output format. 'json' or custom template").
		Short('o').Default("sdfg").String()

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	p, err := newPrinter(os.Stdout, *format)
	if err != nil {
		fmt.Printf("fatal err: %v\n", err)
		os.Exit(1)
	}

	var g run.Group
	{
		ctx, cancel := context.WithCancel(context.Background())
		var (
			execute func() error
		)

		switch cmd {
		case targetsCmd.FullCommand():

			execute = func() error {
				return evalTargets(ctx, p, *configFile, *configMapItem)
			}
		}
		g.Add(execute, func(error) {
			cancel()
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(cancel)
		}, func(error) {
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		p.Printf("fatal err: %v\n", err)
		os.Exit(1)
	}
	p.Printf("\nCommand finished successfully\n")
	os.Exit(0)
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-c:
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}

func loadConfig(configFile string, configMapItem string) (*config.Config, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	cfgStr := string(b)
	if configMapItem != "" {
		// User specified that given configFile is configmap.
		var c v1.ConfigMap

		err = yaml.Unmarshal(b, &c)
		if err != nil {
			return nil, errors.Wrap(err, "load config file as JSON configmap")
		}

		var ok bool
		cfgStr, ok = c.Data[configMapItem]
		if !ok {
			return nil, errors.Wrapf(err, "no data item %s in given configmap", configMapItem)
		}
	}

	return config.Load(cfgStr)
}

func evalTargets(ctx context.Context, p *printer, configFile string, configMapItem string) error {
	p.Printf("Starting jobs command\n")

	cfg, err := loadConfig(configFile, configMapItem)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	targets, err := targets(ctx, p, cfg)
	if err != nil {
		return err
	}

	return p.Print(targets)
}

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ghodss/yaml"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/relabel"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/pkg/api/v1"
)

type executor func(ctx context.Context) func() error

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Playground for Prometheus scrape configuration.")
	app.Version(version.Print("promeval"))
	app.HelpFlag.Short('h')

	var (
		targetExecutor, relabelExecutor executor
	)

	targetsCmd := app.Command("targets", "Evaluate and prints all jobs' scrape targets from given configuration file.")
	{
		configFile := targetsCmd.Arg(
			"config-file",
			"The Prometheus config file to evaluate.",
		).Required().ExistingFile()
		configMapItem := targetsCmd.Flag(
			"configmap-item",
			"If specified, config-file will be treated as configmap with Prometheus config as item under this key.",
		).String()
		job := targetsCmd.Flag(
			"job",
			"If specified, only job with given name will be processed.",
		).String()
		source := targetsCmd.Flag(
			"source",
			"If specified, only targets from specified source will be processed. (you can print all the sources using -o '{{.Source}}' )",
		).String()
		format := targetsCmd.Flag("output", "output format. 'json', 'yaml' or custom template").
			Short('o').Default("").String()

		targetExecutor = func(ctx context.Context) func() error {
			return func() error {
				p, err := newPrinter(os.Stdout, *format)
				if err != nil {
					fmt.Printf("fatal err: %v\n", err)
					os.Exit(1)
				}
				return evalTargets(ctx, p, *configFile, *configMapItem, *job, *source)
			}
		}
	}

	relabelCmd := app.Command("relabel", "Relabel given labels using scrape job's (both target and metric) relabel configs from given configuration file.")
	{
		configFile := relabelCmd.Arg(
			"config-file",
			"The Prometheus config file to fetch relabel configuration from.",
		).Required().ExistingFile()
		configMapItem := relabelCmd.Flag(
			"configmap-item",
			"If specified, config-file will be treated as configmap with Prometheus config as item under this key.",
		).String()
		job := relabelCmd.Flag(
			"job",
			"Scrape job to fetch relabel configs from.",
		).Required().String()
		incTargetRelabel := relabelCmd.Flag(
			"include-target-relabel",
			"If true target relabel configs will be included in process.",
		).Default("true").Bool()
		incMetricRelabel := relabelCmd.Flag(
			"include-metric-relabel",
			"If true metric relabel configs will be included in process.",
		).Default("true").Bool()
		labels := relabelCmd.Flag(
			"label",
			"Labels as an input for relabel (repeated flag).",
		).Required().Strings()
		format := relabelCmd.Flag("output", "output format. 'json', 'yaml' or custom template").
			Short('o').Default("").String()

		relabelExecutor = func(ctx context.Context) func() error {
			return func() error {
				p, err := newPrinter(os.Stdout, *format)
				if err != nil {
					fmt.Printf("fatal err: %v\n", err)
					os.Exit(1)
				}
				return evalRelabel(ctx, p, *configFile, *configMapItem, *job, *labels, *incTargetRelabel, *incMetricRelabel)
			}
		}
	}

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	var g run.Group
	{
		ctx, cancel := context.WithCancel(context.Background())
		var (
			execute func() error
		)

		switch cmd {
		case targetsCmd.FullCommand():
			execute = targetExecutor(ctx)
		case targetsCmd.FullCommand():
			execute = relabelExecutor(ctx)
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
		fmt.Printf("fatal err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nCommand finished successfully\n")
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

func evalTargets(ctx context.Context, p *printer, configFile string, configMapItem string, job string, source string) error {
	p.Printf("Starting targets command\n")

	cfg, err := loadConfig(configFile, configMapItem)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	jobFilter := nopJobFilter
	if job != "" {
		jobFilter = func(jobs []*config.ScrapeConfig) []*config.ScrapeConfig {
			var filtered []*config.ScrapeConfig
			for _, j := range jobs {
				if j.JobName != job {
					continue
				}
				filtered = append(filtered, j)
			}
			return filtered
		}
	}

	sourceFilter := nopSourceFilter
	if source != "" {
		sourceFilter = func(sources map[string]*config.TargetGroup) map[string]*config.TargetGroup {
			filtered := map[string]*config.TargetGroup{}
			for sn, s := range sources {
				if s.Source != source {
					continue
				}
				filtered[sn] = s
			}
			return filtered
		}
	}

	targets, err := targets(ctx, p, cfg, jobFilter, sourceFilter)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		return p.Printf("No targets found. \n")
	}

	return p.Print(targets)
}

func evalRelabel(_ context.Context, p *printer, configFile string, configMapItem string, job string, labels []string, incTargetRelabel bool, incMetricRelabel bool) error {
	p.Printf("Starting targets command\n")

	cfg, err := loadConfig(configFile, configMapItem)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	lset, err := parseFlagLabels(labels)
	if err != nil {
		return errors.Wrap(err, "parse labels")
	}

	var found *config.ScrapeConfig
	for _, scfg := range cfg.ScrapeConfigs {
		if scfg.JobName != job {
			continue
		}

		found = scfg
		break
	}

	if found == nil {
		return errors.Errorf("scrape job %s not found", job)
	}

	var relabelCfg []*config.RelabelConfig
	if incTargetRelabel {
		for _, rcfgs := range found.RelabelConfigs {
			relabelCfg = append(relabelCfg, rcfgs)
		}
	}

	if incMetricRelabel {
		for _, rcfgs := range found.MetricRelabelConfigs {
			relabelCfg = append(relabelCfg, rcfgs)
		}
	}

	if len(relabelCfg) == 0 {
		return errors.Errorf("no relabel config found in job %s. TargetRelabel configs included: %v, MetricRelabel configs included: %v", job, incTargetRelabel, incMetricRelabel)
	}

	tg := TargetLabels{
		Before: lset,
		After:  relabel.Process(lset, relabelCfg...),
	}

	return p.Print(tg)
}

func parseFlagLabels(s []string) (labels.Labels, error) {
	var lset labels.Labels
	for _, l := range s {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("unrecognized label %q", l)
		}
		val, err := strconv.Unquote(parts[1])
		if err != nil {
			return nil, errors.Wrap(err, "unquote label value")
		}
		lset = append(lset, labels.Label{Name: parts[0], Value: val})
	}
	return lset, nil
}

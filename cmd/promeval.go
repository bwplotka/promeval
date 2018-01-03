package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/pkg/api/v1"
	"github.com/ghodss/yaml"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Playground for Prometheus scrape configuration.")
	app.Version(version.Print("promeval"))
	app.HelpFlag.Short('h')

	targets := app.Command("targets", "Evaluate and prints scrape targets from given configurations.")
	configFile := targets.Arg(
		"config-file",
		"The Prometheus config file to evaluate.",
	).Required().ExistingFile()
	configMapItem := targets.Flag(
		"configmap-item",
		"If specified, config-file will be treated as configmap with Prometheus config as item under this key.",
	).String()

	var (
		p = newPrinter(os.Stdout)
		g run.Group
	)
	{
		ctx, cancel := context.WithCancel(context.Background())
		var (
			execute func() error
		)

		switch kingpin.MustParse(app.Parse(os.Args[1:])) {
		case targets.FullCommand():
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
	p.Printf("Command finished successfully\n")
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
	p.Printf("Starting targets command\n")

	cfg, err := loadConfig(configFile, configMapItem)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	for _, scfg := range cfg.ScrapeConfigs {
		targets := discoverTargets(ctx, p, scfg)
		p.PrintTargetGroups(targets)
	}

	return nil
}

func discoverTargets(ctx context.Context, p *printer, scfg *config.ScrapeConfig) map[string]*config.TargetGroup {
	ctx, cancelProviders := context.WithCancel(ctx)

	var (
		wg           sync.WaitGroup
		mtx          sync.Mutex
		targetGroups = map[string]*config.TargetGroup{}
	)

	providers := discovery.ProvidersFromConfig(scfg.ServiceDiscoveryConfig, p.Logger())
	for name, prov := range providers {
		wg.Add(1)

		updates := make(chan []*config.TargetGroup)
		go prov.Run(ctx, updates)

		go func(name string, prov discovery.TargetProvider) {
			select {
			case <-ctx.Done():
			case initial, ok := <-updates:
				// Handle the case that a target provider exits and closes the channel
				// before the context is done.
				if !ok {
					break
				}
				// First set of all targets the provider knows.
				for _, tgroup := range initial {
					mtx.Lock()
					targetGroups[name+"/"+tgroup.Source] = tgroup
					mtx.Unlock()
				}
			case <-time.After(5 * time.Second):
				// Initial set didn't arrive. Act as if it was empty
				// and wait for updates later on.
			}
			wg.Done()

			// We don't care about further updates.
		}(name, prov)
	}

	// We wait for a full initial set of target groups.
	wg.Wait()
	cancelProviders()
	return targetGroups
}

type printer struct {
	w io.Writer
	l log.Logger
}

func newPrinter(w io.Writer) *printer {
	return &printer{
		w: w,
		l: log.NewJSONLogger(w),
	}
}

func (p *printer) Logger() log.Logger {
	return p.l
}

func (p *printer) Printf(format string, a ...interface{}) {
	fmt.Fprintf(p.w, format, a...)
}

func (p *printer) PrintTargetGroups(tgroups map[string]*config.TargetGroup) {
	//tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	//defer tw.Flush()
	//for _,t := range tgroups {
	//
	//}
	spew.Dump(tgroups)
}

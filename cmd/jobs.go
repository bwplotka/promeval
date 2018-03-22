package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/relabel"
)

type TargetLabels struct {
	Before labels.Labels `json:"before",yaml:"before"`
	After  labels.Labels `json:"after",yaml:"after"`
}

// Target is the smallest elemement. We keep the structure flat for a reason -> easier JSON manipulation.
type Target struct {
	JobName string `json:"job_name",yaml:"job_name"`
	Source  string `json:"source",yaml:"source"`

	TargetLabels
}

type scrapeJobFilter func([]*config.ScrapeConfig) []*config.ScrapeConfig

func nopJobFilter(jobs []*config.ScrapeConfig) []*config.ScrapeConfig {
	return jobs
}

type sourceFilter func(map[string]*config.TargetGroup) map[string]*config.TargetGroup

func nopSourceFilter (sources map[string]*config.TargetGroup) map[string]*config.TargetGroup{
	return sources
}

func targets(ctx context.Context, p *printer, cfg *config.Config, jobFilter scrapeJobFilter, sourceFilter sourceFilter) (targets []Target, err error) {
	dlog := p.DiscoveryLogger()
	for _, scfg := range jobFilter(cfg.ScrapeConfigs) {
		tgroups := discoverGroups(ctx, dlog, scfg)

		for _, g := range sourceFilter(tgroups) {
			targetsLabels, err := targetsFromGroup(g, scfg)
			if err != nil {
				// Partial err?
				return nil, errors.Wrap(err, "Target from groups")
			}

			for _, t := range targetsLabels {
				targets = append(targets, Target{
					JobName:      scfg.JobName,
					Source:       g.Source,
					TargetLabels: t,
				})
			}
		}
	}
	p.Printf("\nErrors:\n")
	dlog.LogAll()
	sort.Slice(targets, func(i, j int) bool {
		cmp := strings.Compare(targets[i].JobName, targets[j].JobName)
		if cmp == 0 {
			cmp = strings.Compare(targets[i].Source, targets[j].Source)
		}

		return cmp < 0
	})

	return targets, nil
}

func discoverGroups(ctx context.Context, l log.Logger, scfg *config.ScrapeConfig) map[string]*config.TargetGroup {
	ctx, cancelProviders := context.WithCancel(ctx)

	var (
		wg           sync.WaitGroup
		mtx          sync.Mutex
		targetGroups = map[string]*config.TargetGroup{}
	)

	providers := discovery.ProvidersFromConfig(scfg.ServiceDiscoveryConfig, l)
	for name, prov := range providers {
		wg.Add(1)

		updates := make(chan []*config.TargetGroup)
		go prov.Run(ctx, updates)

		go func(name string, prov discovery.TargetProvider) {
			select {
			case <-ctx.Done():
			case initial, ok := <-updates:
				// Handle the case that a Target provider exits and closes the channel
				// before the context is done.
				if !ok {
					break
				}
				// First set of all targets the provider knows.
				for _, tgroup := range initial {
					if tgroup == nil {
						// No group.
						fmt.Printf("Provider %s does not return any targets.\n", name)
						continue
					}
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

	// We wait for a full initial set of Target groups.
	wg.Wait()
	cancelProviders()
	return targetGroups
}

func targetsFromGroup(tg *config.TargetGroup, cfg *config.ScrapeConfig) ([]TargetLabels, error) {
	targets := make([]TargetLabels, 0, len(tg.Targets))

	for i, tlset := range tg.Targets {
		lbls := make([]labels.Label, 0, len(tlset)+len(tg.Labels))

		for ln, lv := range tlset {
			lbls = append(lbls, labels.Label{Name: string(ln), Value: string(lv)})
		}
		for ln, lv := range tg.Labels {
			if _, ok := tlset[ln]; !ok {
				lbls = append(lbls, labels.Label{Name: string(ln), Value: string(lv)})
			}
		}

		lset := labels.New(lbls...)

		lbls, origLabels, err := populateLabels(lset, cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "populate %d in group %s", i, tg)
		}
		t := TargetLabels{Before: origLabels}
		if lbls != nil {
			t.After = lbls
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// populateLabels builds a label set from the given label set and scrape configuration.
// It returns a label set before relabeling was applied as the second return value.
// Returns a nil label set if the Target is dropped during relabeling.
func populateLabels(lset labels.Labels, cfg *config.ScrapeConfig) (res, orig labels.Labels, err error) {
	// Copy labels into the labelset for the Target if they are not set already.
	scrapeLabels := []labels.Label{
		{Name: model.JobLabel, Value: cfg.JobName},
		{Name: model.MetricsPathLabel, Value: cfg.MetricsPath},
		{Name: model.SchemeLabel, Value: cfg.Scheme},
	}
	lb := labels.NewBuilder(lset)

	for _, l := range scrapeLabels {
		if lv := lset.Get(l.Name); lv == "" {
			lb.Set(l.Name, l.Value)
		}
	}
	// Encode scrape query parameters as labels.
	for k, v := range cfg.Params {
		if len(v) > 0 {
			lb.Set(model.ParamLabelPrefix+k, v[0])
		}
	}

	preRelabelLabels := lb.Labels()
	lset = relabel.Process(preRelabelLabels, cfg.RelabelConfigs...)

	// Check if the Target was dropped.
	if lset == nil {
		return nil, nil, nil
	}
	if v := lset.Get(model.AddressLabel); v == "" {
		return nil, nil, fmt.Errorf("no address")
	}

	lb = labels.NewBuilder(lset)

	// addPort checks whether we should add a default port to the address.
	// If the address is not valid, we don't append a port either.
	addPort := func(s string) bool {
		// If we can split, a port exists and we don't have to add one.
		if _, _, err := net.SplitHostPort(s); err == nil {
			return false
		}
		// If adding a port makes it valid, the previous error
		// was not due to an invalid address and we can append a port.
		_, _, err := net.SplitHostPort(s + ":1234")
		return err == nil
	}
	addr := lset.Get(model.AddressLabel)
	// If it's an address with no trailing port, infer it based on the used scheme.
	if addPort(addr) {
		// Addresses reaching this point are already wrapped in [] if necessary.
		switch lset.Get(model.SchemeLabel) {
		case "http", "":
			addr = addr + ":80"
		case "https":
			addr = addr + ":443"
		default:
			return nil, nil, fmt.Errorf("invalid scheme: %q", cfg.Scheme)
		}
		lb.Set(model.AddressLabel, addr)
	}

	if err := config.CheckTargetAddress(model.LabelValue(addr)); err != nil {
		return nil, nil, err
	}

	// Meta labels are deleted after relabelling. Other internal labels propagate to
	// the Target which decides whether they will be part of their label set.
	for _, l := range lset {
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			lb.Del(l.Name)
		}
	}

	// Default the instance label to the Target address.
	if v := lset.Get(model.InstanceLabel); v == "" {
		lb.Set(model.InstanceLabel, addr)
	}

	res = lb.Labels()
	for _, l := range res {
		// Check label values are valid, drop the Target if not.
		if !model.LabelValue(l.Value).IsValid() {
			return nil, nil, fmt.Errorf("invalid label value for %q: %q", l.Name, l.Value)
		}
	}
	return res, preRelabelLabels, nil
}

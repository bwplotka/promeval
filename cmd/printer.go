package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type printObj func(v interface{}) error

type printer struct {
	w          io.Writer
	l          log.Logger
	printObjFn printObj
}

func newPrinter(w io.Writer, format string) (*printer, error) {
	var p printObj
	switch format {
	case "", "json":
		p = func(v interface{}) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "\t")
			return enc.Encode(v)
		}
	case "yaml":
		p = func(v interface{}) error {
			o, err := yaml.Marshal(v)
			if err != nil {
				return err
			}
			_, err = w.Write(o)
			return err
		}
	default:
		tmpl, err := template.New("").
			Funcs(map[string]interface{}{
				"marshal": func(v interface{}) string {
					a, _ := json.Marshal(v)
					return string(a)
				},
			}).
			Parse(format)
		if err != nil {
			return nil, errors.Wrap(err, "invalid template")
		}
		p = func(v interface{}) error {
			return tmpl.Execute(w, v)
		}
	}

	return &printer{
		w:          w,
		l:          level.NewFilter(log.NewLogfmtLogger(w), level.AllowWarn()),
		printObjFn: p,
	}, nil
}

type discoveryLogger struct {
	l       log.Logger
	keyvals [][]interface{}
}

func (l *discoveryLogger) Log(keyvals ...interface{}) error {
	for i, kv := range keyvals {
		skv, ok := kv.(string)
		if !ok || skv != "err" || i+1 > (len(keyvals)-1) {
			continue
		}
		skvPlus, ok := keyvals[i+1].(error)
		if !ok || skvPlus.Error() != "context canceled" {
			continue
		}
		// Ignore this log line.
		return nil
	}

	// Aggregate all logs for future use.
	l.keyvals = append(l.keyvals, keyvals)
	return nil
}

func (l *discoveryLogger) LogAll() error {
	for _, kvs := range l.keyvals {
		err := l.l.Log(kvs...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) DiscoveryLogger() *discoveryLogger {
	return &discoveryLogger{l: p.l}
}

func (p *printer) Printf(format string, a ...interface{}) error {
	_, err := fmt.Fprintf(p.w, format, a...)
	return err
}

func (p *printer) Print(a ...interface{}) error {
	for _, obj := range a {
		err := p.printObjFn(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

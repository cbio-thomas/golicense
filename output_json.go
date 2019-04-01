package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/mitchellh/golicense/config"
	"github.com/mitchellh/golicense/license"
	"github.com/mitchellh/golicense/module"
)

// JSONSummary --
type JSONSummary struct {
	Import  string      `json:"import"`
	Version string      `json:"version"`
	License interface{} `json:"license"`
}

// JSONOutput writes the results of license lookups to an XLSX file.
type JSONOutput struct {
	// Path is the path to the file to write. This will be overwritten if
	// it exists.
	Path string

	// Config is the configuration (if any). This will be used to check
	// if a license is allowed or not.
	Config *config.Config

	modules map[*module.Module]interface{}
	lock    sync.Mutex
}

// Start implements Output
func (o *JSONOutput) Start(m *module.Module) {}

// Update implements Output
func (o *JSONOutput) Update(m *module.Module, t license.StatusType, msg string) {}

// Finish implements Output
func (o *JSONOutput) Finish(m *module.Module, l *license.License, err error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.modules == nil {
		o.modules = make(map[*module.Module]interface{})
	}

	o.modules[m] = l
	if err != nil {
		o.modules[m] = err
	}
}

// Close implements Output
func (o *JSONOutput) Close() error {

	o.lock.Lock()
	defer o.lock.Unlock()

	// Sort the modules by name
	keys := make([]string, 0, len(o.modules))
	index := map[string]*module.Module{}
	licenses := map[string]interface{}{}
	// licenses := map[string]*license.License{}
	for m, l := range o.modules {
		keys = append(keys, m.Path)
		index[m.Path] = m

		licenses[m.Path] = l
	}
	sort.Strings(keys)

	final := make([]JSONSummary, len(keys))
	// Go through each module and output it into the spreadsheet
	for i, k := range keys {
		m := index[k]
		l := licenses[k]
		switch t := l.(type) {
		case error:
			final[i] = JSONSummary{
				Import:  m.Path,
				Version: m.Version,
				License: &license.License{
					Name: "not found",
					SPDX: "NOT-FOUND",
				},
			}
		case *license.License:
			final[i] = JSONSummary{
				Import:  m.Path,
				Version: m.Version,
				License: l,
			}
		default:
			return fmt.Errorf("unexpected license type: %T", t)
		}
	}

	jsonb, err := json.MarshalIndent(final, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(o.Path, jsonb, 0644)
}

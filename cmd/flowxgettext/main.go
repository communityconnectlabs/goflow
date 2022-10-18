package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/buger/jsonparser"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/flows/definition"
	"github.com/greatnonprofits-nfp/goflow/flows/definition/migrations"
	"github.com/greatnonprofits-nfp/goflow/flows/translation"
	"github.com/pkg/errors"
)

const usage = `usage: flowxgettext [flags] <flowfile>...`

func main() {
	var excludeArgs bool
	var lang string
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.StringVar(&lang, "lang", "", "translation language to extract")
	flags.BoolVar(&excludeArgs, "exclude-args", false, "whether to exclude localized router arguments")
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) == 0 {
		fmt.Println(usage)
		flags.PrintDefaults()
		os.Exit(1)
	}
	if err := FlowXGetText(envs.Language(lang), excludeArgs, args, os.Stdout); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func FlowXGetText(lang envs.Language, excludeArgs bool, paths []string, writer io.Writer) error {
	sources, err := loadFlows(paths)
	if err != nil {
		return err
	}

	var excludeProperties []string
	if excludeArgs {
		excludeProperties = []string{"arguments"}
	}

	po, err := translation.ExtractFromFlows("Generated by flowxgettext", lang, excludeProperties, sources...)
	if err != nil {
		return err
	}

	po.Write(writer)

	return nil
}

// loads all the flows in the given file paths which may be asset files or single flow definitions
func loadFlows(paths []string) ([]flows.Flow, error) {
	flows := make([]flows.Flow, 0)
	for _, path := range paths {
		fileJSON, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading flow file '%s'", path)
		}

		var flowDefs []json.RawMessage

		flowsSection, _, _, err := jsonparser.Get(fileJSON, "flows")
		if err == nil {
			// file is a set of assets with a flow section
			jsonparser.ArrayEach(flowsSection, func(flowJSON []byte, dataType jsonparser.ValueType, offset int, err error) {
				flowDefs = append(flowDefs, flowJSON)
			})
		} else {
			// file is a single flow definition
			flowDefs = append(flowDefs, fileJSON)
		}

		for _, flowDef := range flowDefs {
			flow, err := definition.ReadFlow(flowDef, &migrations.Config{BaseMediaURL: "http://temba.io"})
			if err != nil {
				return nil, errors.Wrapf(err, "error reading flow '%s'", path)
			}
			flows = append(flows, flow)
		}
	}

	return flows, nil
}

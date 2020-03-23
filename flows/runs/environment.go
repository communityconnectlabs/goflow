package runs

import (
	"regexp"
	"strings"
	"time"

	"github.com/greatnonprofits-nfp/goflow/assets"
	"github.com/greatnonprofits-nfp/goflow/envs"
	"github.com/greatnonprofits-nfp/goflow/flows"
	"github.com/greatnonprofits-nfp/goflow/utils"

	"github.com/pkg/errors"
)

type runEnvironment struct {
	envs.Environment

	run *flowRun
}

// creates a run environment based on the given run
func newRunEnvironment(base envs.Environment, run *flowRun) flows.RunEnvironment {
	return &runEnvironment{base, run}
}

func (e *runEnvironment) Timezone() *time.Location {
	contact := e.run.Contact()

	// if run has a contact with a timezone, that overrides the enviroment's timezone
	if contact != nil && contact.Timezone() != nil {
		return contact.Timezone()
	}
	return e.run.Session().Environment().Timezone()
}

func (e *runEnvironment) Locations() (assets.LocationHierarchy, error) {
	sessionAssets := e.run.Session().Assets()
	hierarchies := sessionAssets.Locations().Hierarchies()
	if len(hierarchies) > 0 {
		// in the future we might support more than one hiearchy per session,
		// but for now we only use the first one
		return hierarchies[0], nil
	}

	return nil, nil
}

// FindLocations returns locations with the matching name (case-insensitive), level and parent (optional)
func (e *runEnvironment) FindLocations(name string, level utils.LocationLevel, parent *utils.Location) ([]*utils.Location, error) {
	locations, err := e.Locations()
	if err != nil {
		return nil, err
	}
	if locations == nil {
		return nil, errors.Errorf("can't find locations in environment which is not location enabled")
	}

	return locations.FindByName(name, level, parent), nil
}

// FindLocationsFuzzy returns matching locations like FindLocations but attempts the following strategies
// to find locations:
//   1. Exact match
//   2. Match with punctuation removed
//   3. Split input into words and try to match each word
//   4. Try to match pairs of words
func (e *runEnvironment) FindLocationsFuzzy(text string, level utils.LocationLevel, parent *utils.Location) ([]*utils.Location, error) {
	// try matching name exactly
	if locations, err := e.FindLocations(text, level, parent); len(locations) > 0 || err != nil {
		return locations, err
	}

	// try with punctuation removed
	stripped := strings.TrimSpace(regexp.MustCompile(`\W+`).ReplaceAllString(text, ""))
	if locations, err := e.FindLocations(stripped, level, parent); len(locations) > 0 || err != nil {
		return locations, err
	}

	// try on each tokenized word
	words := regexp.MustCompile(`\W+`).Split(text, -1)
	for _, word := range words {
		if locations, err := e.FindLocations(word, level, parent); len(locations) > 0 || err != nil {
			return locations, err
		}
	}

	// try with each pair of words
	for i := 0; i < len(words)-1; i++ {
		wordPair := strings.Join(words[i:i+2], " ")
		if locations, err := e.FindLocations(wordPair, level, parent); len(locations) > 0 || err != nil {
			return locations, err
		}
	}

	return []*utils.Location{}, nil
}

func (e *runEnvironment) LookupLocation(path utils.LocationPath) (*utils.Location, error) {
	locations, err := e.Locations()
	if err != nil {
		return nil, err
	}
	if locations == nil {
		return nil, errors.Errorf("can't lookup locations in environment which is not location enabled")
	}

	return locations.FindByPath(path), nil
}

var _ flows.RunEnvironment = (*runEnvironment)(nil)

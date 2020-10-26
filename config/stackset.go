package config

import (
	"regexp"
)

type StacksDB struct {
	All    []*StackConfig
	byName map[string]*StackConfig
	byARN  map[string]*StackConfig
}

func (s *StacksDB) AddStack(stacks ...*StackConfig) {
	if s.byName == nil {
		s.byName = map[string]*StackConfig{}
	}

	if s.byARN == nil {
		s.byARN = map[string]*StackConfig{}
	}

	for _, stack := range stacks {
		if stack.Name != "unknown" && stack.Name != "" {
			s.byName[stack.Name] = stack
		}

		s.byARN[stack.ARN] = stack
		s.All = append(s.All, stack)
	}
}

func (s *StacksDB) FindByName(name string) *StackConfig {
	return s.byName[name]
}

func (s *StacksDB) FindByARN(name string) *StackConfig {
	return s.byARN[name]
}

func (s StacksDB) Len() int {
	return len(s.All)
}

// TODO: this should return a new (smaller) StacksDB
func (s *StacksDB) Filter(keys ...string) ([]*StackConfig, error) {
	// TODO: dedup here
	res := []*StackConfig{}

	if len(keys) == 0 {
		return s.All, nil
	}

	for _, k := range keys {
		r, err := regexp.Compile(k)
		if err != nil {
			return nil, err
		}

		for _, stack := range s.All {
			if r.MatchString(stack.Name) {
				res = append(res, stack)
			}
		}
	}

	return res, nil
}

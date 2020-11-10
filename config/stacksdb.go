package config

import (
	"log"
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
		if _, ok := s.byARN[stack.Name]; ok {
			log.Printf("Warning: Already added %q skipping", stack.Name)
			continue
		}

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

func (s *StacksDB) Filter(keys ...string) (*StacksDB, error) {
	res := &StacksDB{}

	if len(keys) == 0 {
		return s, nil
	}

	for _, k := range keys {
		r, err := regexp.Compile(k)
		if err != nil {
			return nil, err
		}

		for _, stack := range s.All {
			if r.MatchString(stack.Name) || r.MatchString(stack.ARN) {
				res.AddStack(stack)
			}
		}
	}

	return res, nil
}

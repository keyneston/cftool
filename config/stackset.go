package config

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

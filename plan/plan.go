package plan

import (
	"encoding/json"
	"os"

	"github.com/keyneston/cftool/config"
)

type Plan struct {
	ChangeSetIDs []string `json:"change_set_ids"`
}

func CreatePlan(file string, stack *config.StackConfig) (*Plan, error) {
	return nil, nil
}

func LoadPlan(file string) (*Plan, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	plan := &Plan{}
	if err := json.NewDecoder(f).Decode(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func (p Plan) Save(file string) error {
	f, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0x644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(p)
}

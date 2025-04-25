package pbpk

import (
	"fmt"
	"os"
	"path/filepath"
	"precisiondosing-api-go/cfg"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ModelDefinition struct {
	ID           string   `yaml:"id"`
	Victim       string   `yaml:"victim"`
	Perpetrators []string `yaml:"perpetrators"`
}

type Models struct {
	Definitions []ModelDefinition
	MaxDoses    int
}

func MustParseAll(config cfg.Models) *Models {
	folder := config.Path
	var models []ModelDefinition

	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() == "models.yaml" {
			models = append(models, mustParseYAML(path)...)
		}

		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("cannot read PBPK model config folder: %v", err))
	}

	result := &Models{
		Definitions: models,
		MaxDoses:    config.MaxDoses,
	}

	return result
}

func mustParseYAML(configFile string) []ModelDefinition {
	f, err := os.Open(configFile)
	if err != nil {
		panic(fmt.Sprintf("cannot open PBPK model config file: %v", err))
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var root map[string]map[string]interface{}
	err = decoder.Decode(&root)
	if err != nil {
		panic(fmt.Sprintf("cannot decode PBPK model config file: %v", err))
	}

	var modelsWrapper struct {
		Models []ModelDefinition `yaml:"models"`
	}

	for _, v := range root {
		yamlBytes, errRoot := yaml.Marshal(v)
		if errRoot != nil {
			panic(fmt.Sprintf("cannot re-marshal nested YAML: %v", errRoot))
		}

		errRoot = yaml.Unmarshal(yamlBytes, &modelsWrapper)
		if errRoot != nil {
			panic(fmt.Sprintf("cannot unmarshal models section: %v", errRoot))
		}

		break
	}

	for i := range modelsWrapper.Models {
		modelsWrapper.Models[i].Victim = strings.ToLower(modelsWrapper.Models[i].Victim)
		for j := range modelsWrapper.Models[i].Perpetrators {
			modelsWrapper.Models[i].Perpetrators[j] = strings.ToLower(modelsWrapper.Models[i].Perpetrators[j])
		}
		sort.Strings(modelsWrapper.Models[i].Perpetrators)
	}

	return modelsWrapper.Models
}

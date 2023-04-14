package helmhog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meln5674/gosh"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	V1Alpha1APIVersion = "helm-hog.meln5674.github.com/v1alpha1"
	ProjectKind        = "Project"

	TempDirParts = "parts"
)

type PartsDirectory = string

type PartPath = string

type Project struct {
	metav1.TypeMeta
	Chart         string                    `json:"chart,omitempty"`
	PartsDirs     []PartsDirectory          `json:"partsDirs,omitempty"`
	Parts         map[PartName]Part         `json:"parts,omitempty"`
	Variables     map[VariableName]Variable `json:"variables"`
	VariableOrder []VariableName            `json:"variableOrder"`
	Requirements  map[RuleName]Requirement  `json:"requirements,omitempty"`
	Restrictions  map[RuleName]Restriction  `json:"restrictions,omitempty"`
}

func (p *Project) Allows(c Case) bool {
	for _, rule := range p.Requirements {
		if !rule.Allows(c) {
			return false
		}
	}
	for _, rule := range p.Restrictions {
		if !rule.Allows(c) {
			return false
		}
	}
	return true
}

func (p *Project) Load() (*LoadedProject, error) {
	var err error
	if p.APIVersion != V1Alpha1APIVersion {
		return nil, fmt.Errorf("Unknown apiVersion: %s", p.APIVersion)
	}
	if p.Kind != ProjectKind {
		return nil, fmt.Errorf("Unknown kind: %s", p.Kind)
	}

	if len(p.PartsDirs)+len(p.Parts) == 0 {
		return nil, fmt.Errorf("No parts or parts directories specified")
	}
	if len(p.Variables) == 0 {
		return nil, fmt.Errorf("No variables specified")
	}
	for name, v := range p.Variables {
		if len(v) == 0 {
			return nil, fmt.Errorf("Variable %s has no choices", name)
		}
	}

	for name, rule := range p.Requirements {
		if len(rule.If) == 0 {
			return nil, fmt.Errorf("Requirement %s has an empty 'if', it will match all cases", name)
		}
		if len(rule.Then) == 0 {
			return nil, fmt.Errorf("Requirment %s has an empty 'then', it will never discard any cases", name)
		}
	}
	for name, rule := range p.Restrictions {
		if len(rule) == 0 {
			return nil, fmt.Errorf("Restriction %s is empty, it will discard all cases", name)
		}
	}

	l := LoadedProject{Project: p}

	l.TempDir, err = os.MkdirTemp("", "helm-hog-*")
	if err != nil {
		return nil, errors.Wrap(err, "create project temp directory")
	}
	defer func() {
		if err != nil {
			os.RemoveAll(l.TempDir)
		}
	}()

	l.PartsMapping = make(map[PartName]PartPath)
	for name, part := range p.Parts {
		if _, ok := l.PartsMapping[name]; ok {
			return nil, fmt.Errorf("Part name %s is duplicated", name)
		}
		err = func() error {
			path := filepath.Join(l.TempDir, name+".yaml")
			partBytes, err := yaml.Marshal(part)
			if err != nil {
				return errors.Wrap(err, "yaml marshal %s")
			}
			f, err := os.Create(path)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("create %s", path))
			}
			defer f.Close()
			_, err = f.Write(partBytes)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("write %s", path))
			}
			l.PartsMapping[name] = path
			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Create temp file for part %s", name))
		}
	}

	for _, dir := range p.PartsDirs {
		err = func() error {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("list directory %s", dir))
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				path := filepath.Join(dir, name)
				if strings.HasSuffix(name, ".yml") {
					name = strings.TrimSuffix(name, ".yml")
				} else if strings.HasSuffix(name, ".yaml") {
					name = strings.TrimSuffix(name, ".yaml")
				} else {
					continue
				}
				if _, ok := l.PartsMapping[name]; ok {
					return fmt.Errorf("Part name %s is duplicated", name)
				}
				l.PartsMapping[name] = path
			}

			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("scan parts dir %s", dir))
		}
	}

	for varName, choices := range p.Variables {
		err = func() error {
			for choiceName, parts := range choices {
				for _, part := range parts {
					if _, ok := l.PartsMapping[part]; !ok {
						return fmt.Errorf("Variable %s, Choice %s refers to an unresolved part %s", varName, choiceName, part)
					}
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	for ruleName, rule := range p.Requirements {
		err = func() error {
			for varName, choiceName := range rule.If {
				if _, ok := p.Variables[varName]; !ok {
					return fmt.Errorf("Requirement %s If refers to undefined variable %s", ruleName, varName)
				}
				if _, ok := p.Variables[varName][choiceName]; !ok {
					return fmt.Errorf("Requirment %s If for variable %s refers to non-existent choice %s", ruleName, varName, choiceName)
				}
			}
			for varName, choiceName := range rule.Then {
				if _, ok := p.Variables[varName]; !ok {
					return fmt.Errorf("Requirement %s Then refers to undefined variable %s", ruleName, varName)
				}
				if _, ok := p.Variables[varName][choiceName]; !ok {
					return fmt.Errorf("Requirment %s Then for variable %s refers to non-existent choice %s", ruleName, varName, choiceName)
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	for ruleName, rule := range p.Restrictions {
		err = func() error {
			for varName, choiceName := range rule {
				if _, ok := p.Variables[varName]; !ok {
					return fmt.Errorf("Restriction %s refers to undefined variable %s", ruleName, varName)
				}
				if _, ok := p.Variables[varName][choiceName]; !ok {
					return fmt.Errorf("Restriction %s for variable %s refers to non-existent choice %s", ruleName, varName, choiceName)
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	l.VariableOrder = make([]VariableName, 0, len(p.Variables))
	if len(p.VariableOrder) == 0 {
		for name := range p.Variables {
			l.VariableOrder = append(l.VariableOrder, name)
		}
	} else {
		missingVariables := make(map[VariableName]struct{}, len(p.Variables))
		for name := range p.Variables {
			missingVariables[name] = struct{}{}
		}
		for _, name := range p.VariableOrder {
			delete(missingVariables, name)
			l.VariableOrder = append(l.VariableOrder, name)
		}
		if len(missingVariables) != 0 {
			// TODO: Nicer error message
			err = fmt.Errorf("variableOrder is missing the following variables: %v", missingVariables)
			return nil, err
		}
	}
	l.ReverseVariableOrder = make([]VariableName, len(p.Variables))
	for ix := range l.VariableOrder {
		l.ReverseVariableOrder[ix] = l.VariableOrder[ix]
	}
	sort.Reverse(sort.StringSlice(l.ReverseVariableOrder))

	if p.Chart == "" {
		l.Chart = "."
	} else {
		l.Chart = p.Chart
	}

	return &l, nil
}

type LoadedProject struct {
	*Project
	TempDir string
	Chart   string

	VariableOrder        []VariableName
	ReverseVariableOrder []VariableName

	PartsMapping map[PartName]PartPath
}

func (l *LoadedProject) GenerateCases(cases chan<- Case) {
	outgoing := cases
	for _, name := range l.ReverseVariableOrder {
		choices := l.Variables[name]
		incoming := make(chan Case)
		go func(name string, choices Variable, incoming <-chan Case, outgoing chan<- Case) {
			for c := range incoming {
				for choice := range choices {
					newC := c.With(name, choice)
					if !l.Allows(newC) {
						continue
					}
					outgoing <- newC
				}
			}
			close(outgoing)
		}(name, choices, incoming, outgoing)
		outgoing = incoming
	}

	outgoing <- Case{}
	close(outgoing)
}

func (l *LoadedProject) ValuesArgs(c Case) []string {
	args := []string{}
	for name, choice := range c {
		for _, part := range l.Variables[name][choice] {
			args = append(args, "--values", l.PartsMapping[part])
		}
	}
	return args
}

func (l *LoadedProject) Lint(c Case) gosh.Commander {
	cmd := []string{"helm", "lint", l.Chart}
	cmd = append(cmd, l.ValuesArgs(c)...)
	return gosh.Command(cmd...).WithStreams(gosh.FileOut(l.TempPath(c, "lint.out")), gosh.FileErr(l.TempPath(c, "lint.err")))
}

func (l *LoadedProject) ApplyDryRun(c Case) gosh.Commander {
	cmd := []string{"helm", "template", l.Chart, "--debug"}
	cmd = append(cmd, l.ValuesArgs(c)...)
	return gosh.Pipeline(
		gosh.Command(cmd...).WithStreams(gosh.FileErr(l.TempPath(c, "template.err"))),
		gosh.Command("tee", l.TempPath(c, "template.out")),
		gosh.Command("kubectl", "apply", "-f", "-", "--dry-run", "client").WithStreams(gosh.FileOut(l.TempPath(c, "apply.out")), gosh.FileErr(l.TempPath(c, "apply.err"))),
	)
}

func (l *LoadedProject) Validate(c Case) gosh.Commander {
	return gosh.And(l.Lint(c), l.ApplyDryRun(c))
}

func (l *LoadedProject) CaseTempDirParts(c Case) []string {
	parts := []string{l.TempDir, "reports"}
	for _, name := range l.VariableOrder {
		parts = append(parts, name, c[name])
	}
	return parts
}

func (l *LoadedProject) MakeCaseTempDir(c Case) error {
	return os.MkdirAll(filepath.Join(l.CaseTempDirParts(c)...), 0700)
}

func (l *LoadedProject) TempPath(c Case, basename string) string {
	parts := l.CaseTempDirParts(c)
	parts = append(parts, basename)
	return filepath.Join(parts...)
}

package website

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	"github.com/trueforge-org/forgetool/pkg/helper"
)

const (
	bakeFile    = "docker-bake.hcl"
	defaultIcon = "https://github.com/trueforge-org/website/blob/main/shared/public/logo.svg"
)

var marshalContainerList = json.Marshal

type ContainerList struct {
	TotalCount int64       `json:"totalCount"`
	Containers []Container `json:"containers"`
}

type Container struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	License string `json:"license"`
	Icon    string `json:"icon"`
}

type ContainerListOptions struct {
	OutputPath string
	sync.Mutex
	list *ContainerList
}

func (o *ContainerListOptions) WriteContainerList() error {
	if o.list == nil {
		return fmt.Errorf("container list is nil")
	}

	data, err := marshalContainerList(o.list)
	if err != nil {
		return err
	}

	return os.WriteFile(o.OutputPath, data, 0644)
}

func (o *ContainerListOptions) GetContainerData(path string, entry os.DirEntry, err error) error {
	if o.list == nil {
		o.list = &ContainerList{}
	}

	if walkErr := validateEntry(entry, err); walkErr != nil {
		return walkErr
	}
	if entry.Name() != bakeFile {
		return nil
	}

	vars, parseErr := parseBakeVariables(path)
	if parseErr != nil {
		return parseErr
	}

	container := buildContainer(vars)

	o.Lock()
	defer o.Unlock()
	o.list.TotalCount++
	o.list.Containers = append(o.list.Containers, container)

	return filepath.SkipDir
}

func validateEntry(entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if entry.IsDir() && slices.Contains(helper.ExcludedDirs, entry.Name()) {
		return filepath.SkipDir
	}

	return nil
}

// variableRe matches HCL variable block openings like:
//
//	variable "APP" {
var variableRe = regexp.MustCompile(`^\s*variable\s+"(\w+)"\s*\{`)
var defaultRe = regexp.MustCompile(`^\s*default\s*=\s*"(.+)"`)

func parseBakeVariables(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %w", path, err)
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	var currentVar string

	for scanner.Scan() {
		line := scanner.Text()

		if m := variableRe.FindStringSubmatch(line); m != nil {
			currentVar = m[1]
			continue
		}

		if currentVar != "" {
			if m := defaultRe.FindStringSubmatch(line); m != nil {
				vars[currentVar] = m[1]
				currentVar = ""
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return vars, nil
}

func buildContainer(vars map[string]string) Container {
	icon := vars["ICON"]
	if icon == "" {
		icon = defaultIcon
	}

	return Container{
		Name:    vars["APP"],
		Version: vars["VERSION"],
		Source:  vars["SOURCE"],
		License: vars["LICENSE"],
		Icon:    icon,
	}
}

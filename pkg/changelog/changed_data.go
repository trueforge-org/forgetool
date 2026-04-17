package changelog

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

type ChangedData struct {
	mu         *sync.RWMutex     `json:"-"`
	LastCommit string            `json:"last_commit"`
	Charts     map[string]*Chart `json:"charts"`
}

var readChangedDataFileFunc = os.ReadFile
var marshalChangedDataFunc = json.MarshalIndent

type Chart struct {
	Versions       map[string]*Version `json:"versions"`
	SortedVersions []string            `json:"-"` // Used only for rendering
	Name           string              `json:"-"` // Used only for rendering
	Train          string              `json:"-"` // Used only for rendering
}

func (c *Chart) SortVersions(reverse bool) ([]*semver.Version, error) {
	chartVersions := []*semver.Version{}
	for key := range c.Versions {
		semVer, err := semver.NewVersion(key)
		if err != nil {
			return nil, err
		}
		chartVersions = append(chartVersions, semVer)
	}
	// Sort the versions from oldest to newest
	sort.Slice(chartVersions, func(i, j int) bool {
		if reverse {
			return chartVersions[i].GreaterThan(chartVersions[j])
		}
		return chartVersions[i].LessThan(chartVersions[j])
	})

	for _, version := range chartVersions {
		c.SortedVersions = append(c.SortedVersions, version.String())
	}

	return chartVersions, nil
}

func (c *ChangedData) AddOrUpdateChart(chart string, version string, train string, commit *object.Commit) {
	if c.Charts == nil {
		c.Charts = make(map[string]*Chart)
	}
	_, exists := c.Charts[chart]
	if !exists {
		c.Charts[chart] = &Chart{}
	}

	c.Charts[chart].AddVersion(version, train)
	c.Charts[chart].Versions[version].AddCommit(commit)
}

func (c *Chart) AddVersion(version string, train string) {
	if c.Versions == nil {
		c.Versions = make(map[string]*Version)
	}
	_, exists := c.Versions[version]
	if exists {
		return
	}
	c.Versions[version] = &Version{
		Version: version,
		Train:   train,
		Commits: make(map[string]*Commit),
	}
}

type Version struct {
	Version       string             `json:"version"`
	Train         string             `json:"train"`
	Commits       map[string]*Commit `json:"commits"`
	SortedCommits []*Commit          `json:"-"` // Used only for rendering
}

func (v *Version) AddCommit(commit *object.Commit) {
	if v.Commits == nil {
		v.Commits = make(map[string]*Commit)
	}

	_, exists := v.Commits[commit.Hash.String()]
	if exists {
		return
	}
	v.Commits[commit.Hash.String()] = &Commit{
		CommitHash: commit.Hash.String(),
		ParentHash: commit.ParentHashes[0].String(),
		Author:     Author{Name: commit.Author.Name, Date: commit.Author.When.Format(dateFormat)},
		Message:    getCommitMessage(commit),
		Kind:       getCommitKind(commit),
	}
}

func (v *Version) SortCommits(reverse bool) ([]*Commit, error) {
	commits := []*Commit{}
	for _, commit := range v.Commits {
		commits = append(commits, commit)
	}

	parsedDates := make(map[*Commit]time.Time, len(commits))
	for _, commit := range commits {
		parsedDate, err := time.Parse(dateFormat, commit.Author.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date [%s]: %w", commit.Author.Date, err)
		}
		parsedDates[commit] = parsedDate
	}

	sort.Slice(commits, func(i, j int) bool {
		iDate := parsedDates[commits[i]]
		jDate := parsedDates[commits[j]]
		if reverse {
			return iDate.After(jDate)
		}
		return iDate.Before(jDate)
	})

	v.SortedCommits = commits
	return commits, nil
}

type Commit struct {
	CommitHash string `json:"commit_hash"`
	ParentHash string `json:"parent_hash"`
	Author     Author `json:"author"`
	Kind       string `json:"kind"`
	Message    string `json:"message"`
}

type Author struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

func (c *ChangedData) LoadFromFile(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory")
	}

	bytes, err := readChangedDataFileFunc(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return err
	}

	return nil
}

func (c *ChangedData) WriteToFile(path string) error {
	data, err := marshalChangedDataFunc(c, "", "  ")
	if err != nil {
		return err
	}
	log.Info().Msgf("Writing changed data to [%s]", path)
	return os.WriteFile(path, data, 0644)
}

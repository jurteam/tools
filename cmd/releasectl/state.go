package main

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type State struct {
	Ver      string `json:"version"`
	Base     string `json:"base"`
	Previous string `json:"previous,omitempty"`
	NumRC    int    `json:"num_preleases"`
}

func NewState(ver, base string) *State {
	if !semver.IsValid(normalizeSemver(ver)) {
		return nil
	}

	return &State{
		Ver:  ver,
		Base: base,
	}
}

func (s *State) Version() string        { return semver.Canonical(s.Ver) }
func (s *State) ReleaseCandidate() bool { return semver.Prerelease(s.Ver) == "" }
func (s *State) Branch() string {
	return strings.Join([]string{flagRelBranchPrefix, trimSemverPrefix(s.Version())}, "/")
}

func (s *State) releaseBranchRefName() string {
	return fmt.Sprintf("refs/heads/%s", s.Branch())
}

func (s *State) CreateReleaseBranch(r *git.Repository) error {
	headRef, err := r.Head()
	if err != nil {
		return fmt.Errorf("CreateReleaseBranch: %v", err)
	}

	refName := s.releaseBranchRefName()
	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), headRef.Hash())

	if err := r.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("CreateReleaseBranch: %v", err)
	}

	if err := r.CreateBranch(&config.Branch{Name: s.Branch()}); err != nil {
		return fmt.Errorf("CreateReleaseBranch: %v", err)
	}

	return nil
}

func (s *State) DeleteBranch(r *git.Repository) error {
	if err := r.Storer.RemoveReference(plumbing.ReferenceName(s.releaseBranchRefName())); err != nil {
		return fmt.Errorf("DeleteBranch: %v", err)
	}

	if err := r.DeleteBranch(s.Branch()); err != nil {
		return fmt.Errorf("DeleteBranch: %v", err)
	}

	return nil
}

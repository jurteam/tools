package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	ReleaseStateFile           = ".release.json"
	DefaultReleaseBranchPrefix = "release"
)

var (
	cwd           string
	stateFilePath string

	flagRelBranchPrefix string
	flagBase            string
)

func init() {
	var err error

	cwd, err = os.Getwd()
	panicIfError(err)

	stateFilePath = path.Join(cwd, ReleaseStateFile)

	flag.StringVar(&flagRelBranchPrefix, "prefix", DefaultReleaseBranchPrefix, "release branch prefix")
	flag.StringVar(&flagBase, "b", "", "branch off which the release is cut; it takes the current branch by default")
}

func main() {
	flag.CommandLine.SetOutput(os.Stderr)
	flag.CommandLine.Usage = displayUsage
	log.SetFlags(0)
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("releasectl: ")
	log.SetOutput(os.Stdout)
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("insufficient arguments")
	}

	switch flag.Arg(0) {
	case "new":
		ensureCmdArgs("new", 1)
		cmdNew(flag.Arg(1))
	case "abort":
		ensureCmdArgs("abort", 0)
		cmdAbort()
	case "rc":
		ensureCmdArgs("rc", 0)
		cmdTagReleaseCandidate()
	case "finalize":
		ensureCmdArgs("finalize", 0)
		cmdFinalize()
	}

}

func cmdNew(v string) {
	v = normalizeSemver(v)
	if !semver.IsValid(v) {
		log.Fatalf("not a valid semantic version: %s", v)
	}

	ensureStateFileNotExist()
	r := ensureRepository()

	// ensure the new release's tag is unique
	ensureTagNotExist(r, v)

	// create release branch
	st := NewState(v, baseBranch(r))

	log.Printf("Initialising new release: %s", st.Version())
	log.Printf("Creating release branch: %s", st.Branch())
	if err := st.CreateReleaseBranch(r); err != nil {
		log.Fatalf("couldn't create the release branch: %v", err)
	}

	serializeState(st)

	log.Printf(
		"Push the release branch to kick off the release process: git push origin -u origin %s", st.Branch())
}

func cmdAbort() {
	ensureStateFileExist()
	r := ensureRepository()
	st := deserializeState()

	log.Printf("Aborting release: %s", st.Version())

	var retCode int

	log.Printf("Deleting release branch: %s", st.Branch())
	if err := st.DeleteBranch(r); err != nil {
		log.Printf("couldn't remove the branch %s: %v", st.Branch(), err)
		retCode = 2
	}

	log.Printf("Prune state file: %s", stateFilePath)
	if err := os.Remove(stateFilePath); err != nil {
		log.Printf("couldn't remove the file %s: %v", st.Branch(), err)
		retCode = 2
	}

	os.Exit(retCode)
}

func cmdTagReleaseCandidate() {
	ensureStateFileExist()
	r := ensureRepository()
	st := deserializeState()

	cwb := currentWorkingBranch(r)
	if cwb != st.Branch() {
		log.Fatalf("cmdTagReleaseCandidate: checkout the branch %s and run this command again", st.Branch())
	}

	rcTag := fmt.Sprintf("%s-pre%d", st.Version(), st.NumRC+1)
	ensureTagNotExist(r, rcTag)
	head, err := r.Head()
	checkError(err)

	_, err = r.CreateTag(rcTag, head.Hash(), nil)
	checkError(err)

	st.NumRC++
	serializeState(st)

	log.Printf("Release Candidate tag created: %s", rcTag)
	log.Printf("Push the changes to update the remote and propagate the changes: git push --tags")
}

func cmdFinalize() {
	ensureStateFileExist()
	r := ensureRepository()
	st := deserializeState()

	log.Printf("Finalizing release %s", st.Version())
	cwb := currentWorkingBranch(r)
	log.Printf("Current working branch: %s", cwb)

	if cwb != st.Branch() {
		log.Fatalf("checkout the branch %s and run this command again", st.Branch())
	}

	releaseTag := st.Version()
	ensureTagNotExist(r, releaseTag)

	head, err := r.Head()
	checkError(err)

	log.Printf("Creating annotated tag %q on commit %q", releaseTag, head.Hash())
	_, err = r.CreateTag(releaseTag, head.Hash(), &git.CreateTagOptions{
		Message: fmt.Sprintf("Release %v", releaseTag)})
	checkError(err)

	log.Printf("Release tag %s created, cleaning up now.", releaseTag)
	cleanup(r, st)
}

func cleanup(r *git.Repository, st *State) {
	w, err := r.Worktree()
	checkError(err)

	revHash, err := r.ResolveRevision(plumbing.Revision("origin/main"))
	checkError(err)

	fmt.Println("ce")
	err = w.Checkout(&git.CheckoutOptions{
		Hash:   plumbing.NewHash(revHash.String()),
		Branch: "main",
		Force:  true,
		Keep:   false,
		Create: true,
	})
	checkError(err)

	if err := st.DeleteBranch(r); err != nil {
		log.Printf("couldn't remove the branch %s: %v", st.Branch(), err)
	}

	if err := os.Remove(stateFilePath); err != nil {
		log.Printf("couldn't remove the file %s: %v", st.Branch(), err)
	}
}

func serializeState(st *State) {
	b, err := json.Marshal(*st)
	checkError(err)
	checkError(os.WriteFile(stateFilePath, b, 0644))
}

func deserializeState() *State {
	var st State
	b, err := os.ReadFile(stateFilePath)
	checkError(err)
	checkError(json.Unmarshal(b, &st))

	if !semver.IsValid(st.Version()) {
		log.Fatalf("deserializeState: %s is not a valid semantic version string", st.Ver)
	}

	return &st
}

func baseBranch(r *git.Repository) string {
	if flagBase == "develop" || flagBase == "staging" {
		return flagBase
	} else if flagBase != "" {
		log.Fatalf("baseBranch: invalid branch %q: %v", flagBase, errInvalidBranch)
	}

	cwb := currentWorkingBranch(r)

	if cwb != "develop" && cwb != "staging" {
		log.Fatalf("baseBranch: currently on %q: %v", cwb, errInvalidBranch)
	}

	return cwb
}

func currentWorkingBranch(r *git.Repository) string {
	head, err := r.Head()
	checkError(err)

	if !head.Name().IsBranch() {
		return ""
	}

	return head.Name().Short()
}

func ensureRepository() *git.Repository {
	r, err := git.PlainOpen(cwd)
	if err != nil {
		log.Fatalf("couldn't find a valid repository: %v", err)
	}

	return r
}

func ensureStateFileNotExist() {
	if _, err := os.Stat(stateFilePath); err != nil && errors.Is(err, os.ErrNotExist) {
		return
	}

	log.Fatal("state file already exists, another release is in progress")
}

func ensureStateFileExist() {
	if _, err := os.Stat(stateFilePath); err != nil && errors.Is(err, os.ErrNotExist) {
		log.Fatal("No release process seems to be ongoing. " +
			"Use the command `releasectl new` to create a new release.")
	}
}

func checkError(e error) {
	if e != nil {
		//panic(e)
		log.Fatal(e)
	}
}

func panicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

func ensureCmdArgs(cmd string, n int) {
	if flag.NArg() != 1+n {
		log.Fatalf("the command %q takes %d arguments (and not %d)", cmd, n, flag.NArg()-1)
	}
}

func normalizeSemver(s string) string {
	if !strings.HasPrefix(s, "v") {
		return fmt.Sprintf("v%s", s)
	}

	return s
}

func ensureTagNotExist(r *git.Repository, v string) {
	tags, err := r.Tags()
	checkError(err)

	err = tags.ForEach(func(t *plumbing.Reference) error {
		if t.Name().Short() == normalizeSemver(v) {
			return fmt.Errorf("the tag already exists: %s", normalizeSemver(v))
		}
		return nil
	})
	checkError(err)
}

func trimSemverPrefix(v string) string { return strings.TrimPrefix(v, "v") }

var errInvalidBranch = fmt.Errorf("releases can be cut off either 'develop' or 'staging' only")

func displayUsage() {
	fmt.Fprintln(flag.CommandLine.Output(), `Usage: releasectl [OPTIONS] COMMAND...
Peerfor.`)
	flag.PrintDefaults()
	fmt.Fprintln(flag.CommandLine.Output(), `
This program assists the release manager in dealing with
the git branching operations and procedures that are
required to be performed by the release process.

The following commands are available:

  abort          Cancel the release process currently ongoing and roll back changes.
  finalize       Finalize and tag the release.
  new            Initiate a new release.
  rc             Create a lightweight tag for a pre-release milestone (RC).

Written by Alessio Treglia <alessio@debian.org>.
`)
}

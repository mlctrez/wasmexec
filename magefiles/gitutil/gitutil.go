package gitutil

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/rogpeppe/go-internal/semver"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type GitUtil struct {
	repo      *git.Repository
	worktree  *git.Worktree
	signature *object.Signature

	semverTags []string
}

func Open(path string) (gu *GitUtil, err error) {
	gu = &GitUtil{}
	if gu.repo, err = git.PlainOpen(path); err != nil {
		return
	}
	if gu.worktree, err = gu.repo.Worktree(); err != nil {
		return
	}
	return
}

func (gu *GitUtil) Signature(name, email string) {
	gu.signature = &object.Signature{Name: name, Email: email, When: time.Now()}
}

func (gu *GitUtil) Add(path, message string) (err error) {

	if gu.signature == nil {
		return fmt.Errorf("call Signature before add")
	}

	if _, err = gu.worktree.Add(path); err != nil {
		return
	}

	opts := &git.CommitOptions{Author: gu.signature}
	if _, err = gu.worktree.Commit(message, opts); err != nil {
		return
	}

	return nil
}

func (gu *GitUtil) PushNewVersion() (err error) {
	var newTag string
	if newTag, err = gu.NextSemverTag(); err != nil {
		return err
	}

	var head *plumbing.Reference
	if head, err = gu.repo.Head(); err != nil {
		return err
	}

	opts := &git.CreateTagOptions{Message: newTag, Tagger: gu.signature}
	if _, err = gu.repo.CreateTag(newTag, head.Hash(), opts); err != nil {
		return err
	}
	specs := []config.RefSpec{
		config.RefSpec(fmt.Sprintf("%s:%s", head.Name(), head.Name())),
		config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", newTag, newTag)),
	}

	token := devToken()
	if token != "" {
		_ = os.Setenv("ACTIONS_TOKEN", token)
	}
	token = os.Getenv("ACTIONS_TOKEN")

	return gu.repo.Push(&git.PushOptions{
		Auth:       &http.BasicAuth{Username: token},
		RemoteName: "origin",
		RefSpecs:   specs,
	})
}

func (gu *GitUtil) fetchTags() (err error) {
	if err = gu.repo.Fetch(&git.FetchOptions{Tags: git.AllTags}); err != nil {
		if "already up-to-date" != err.Error() {
			return err
		}
	}
	var tags storer.ReferenceIter
	tags, err = gu.repo.Tags()
	gu.semverTags = []string{}
	if err = tags.ForEach(func(reference *plumbing.Reference) error {
		if semver.IsValid(reference.Name().Short()) {
			gu.semverTags = append(gu.semverTags, reference.Name().Short())
		}
		return nil
	}); err != nil {
		return err
	}
	sort.SliceStable(gu.semverTags, func(i, j int) bool {
		return semver.Compare(gu.semverTags[i], gu.semverTags[j]) > 0
	})
	return nil
}

func (gu *GitUtil) NextSemverTag() (tag string, err error) {
	if err = gu.fetchTags(); err != nil {
		return "", err
	}

	latest := gu.semverTags[0]
	split := strings.Split(latest, ".")

	var i int
	i, err = strconv.Atoi(split[2])
	if err != nil {
		return "", err
	}

	split[2] = fmt.Sprintf("%s", fmt.Sprintf("%d", i+1))
	tag = fmt.Sprintf("%s.%s.%s", split[0], split[1], split[2])

	return tag, nil
}

func devToken() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	var tokenBytes []byte
	tokenBytes, err = os.ReadFile(filepath.Join(dir, ".github_token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(tokenBytes))
}

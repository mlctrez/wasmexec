package gitutil

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"os"
	"path/filepath"
	"strings"
)

type GitUtil struct {
	origin   string
	repoDir  string
	repo     *git.Repository
	worktree *git.Worktree
}

func New(origin, repoDir string) *GitUtil {
	return &GitUtil{origin: origin, repoDir: repoDir}
}

func (g *GitUtil) CloneOrOpen() (err error) {
	_, err = os.Stat(g.repoDir)

	if os.IsNotExist(err) {
		g.repo, err = git.PlainClone(g.repoDir, false, &git.CloneOptions{
			URL:      g.origin,
			Progress: os.Stdout,
			Depth:    1,
		})
		if err != nil {
			return
		}
	} else {
		if g.repo, err = git.PlainOpen(g.repoDir); err != nil {
			return
		}
	}
	if g.worktree, err = g.repo.Worktree(); err != nil {
		return
	}

	return nil
}

func (g *GitUtil) Tags(prefix string) (refs []*plumbing.Reference, err error) {
	var iter storer.ReferenceIter
	if iter, err = g.repo.Tags(); err != nil {
		return
	}
	err = iter.ForEach(func(reference *plumbing.Reference) error {
		shortName := reference.Name().Short()
		if strings.HasPrefix(shortName, prefix) {
			refs = append(refs, reference)
		}
		return nil
	})
	return
}

func (g *GitUtil) Contents(path string, ref *plumbing.Reference) (content []byte, err error) {
	if err = g.worktree.Reset(&git.ResetOptions{Commit: ref.Hash(), Mode: git.HardReset}); err != nil {
		return
	}
	content, err = os.ReadFile(filepath.Join(g.repoDir, path))
	return
}

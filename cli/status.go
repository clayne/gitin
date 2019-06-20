package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/isacikgoz/gia/editor"
	"github.com/isacikgoz/gitin/prompt"
	"github.com/isacikgoz/gitin/term"
	git "github.com/isacikgoz/libgit2-api"
)

// status holds the repository struct and the prompt pointer.
type status struct {
	repository *git.Repository
	prompt     *prompt.Prompt
}

// StatusPrompt configures a prompt to serve as work-dir explorer prompt
func StatusPrompt(r *git.Repository, opts *prompt.Options) (*prompt.Prompt, error) {
	st, err := r.LoadStatus()
	if err != nil {
		return nil, fmt.Errorf("could not load status: %v", err)
	}
	if len(st.Entities) == 0 {
		writer := term.NewBufferedWriter(os.Stdout)
		for _, line := range workingTreeClean(r.Head) {
			writer.WriteCells(line)
		}
		writer.Flush()
		os.Exit(0)
	}
	list, err := prompt.NewList(st.Entities, opts.LineSize)
	if err != nil {
		return nil, fmt.Errorf("could not create list: %v", err)
	}
	controls := make(map[string]string)
	controls["add/reset entry"] = "space"
	controls["show diff"] = "enter"
	controls["add all"] = "a"
	controls["reset all"] = "r"
	controls["hunk stage"] = "p"
	controls["commit"] = "c"
	controls["amend"] = "m"
	controls["discard changes"] = "!"

	s := &status{repository: r}

	s.prompt = prompt.Create("Files", opts, list,
		prompt.WithKeyHandler(s.onKey),
		prompt.WithSelectionHandler(s.onSelect),
		prompt.WithItemRenderer(renderItem),
		prompt.WithInformation(s.info),
	)
	s.prompt.Controls = controls

	return s.prompt, nil
}

// return err to terminate
func (s *status) onSelect() error {
	item, err := s.prompt.Selection()
	if err != nil {
		return fmt.Errorf("can't show diff: %v", err)
	}
	entry := item.(*git.StatusEntry)
	if err = popGitCommand(s.repository, fileStatArgs(entry)); err != nil {
		return nil // intentionally ignore errors here
	}
	return nil
}

// lots of command handling here
func (s *status) onKey(key rune) error {
	var reqReload bool
	switch key {
	case ' ':
		reqReload = true
		item, err := s.prompt.Selection()
		if err != nil {
			return fmt.Errorf("can't add/reset item: %v", err)
		}
		entry := item.(*git.StatusEntry)
		args := []string{"add", "--", entry.String()}
		if entry.Indexed() {
			args = []string{"reset", "HEAD", "--", entry.String()}
		}
		cmd := exec.Command("git", args...)
		cmd.Dir = s.repository.Path()
		if err := cmd.Run(); err != nil {
			return err
		}
	case 'p':
		reqReload = true
		// defer s.prompt.writer.HideCursor()
		item, err := s.prompt.Selection()
		if err != nil {
			return fmt.Errorf("can't hunk stage item: %v", err)
		}
		entry := item.(*git.StatusEntry)
		file, err := generateDiffFile(s.repository, entry)
		if err == nil {
			editor, err := editor.NewEditor(file)
			if err != nil {
				return err
			}
			patches, err := editor.Run()
			if err != nil {
				return err
			}
			for _, patch := range patches {
				if err := applyPatchCmd(s.repository, entry, patch); err != nil {
					return err
				}
			}
		}
	case 'c':
		reqReload = true
		// defer s.prompt.writer.HideCursor()
		args := []string{"commit", "--edit", "--quiet"}
		err := popGitCommand(s.repository, args)
		if err != nil {
			return err
		}
		s.repository.LoadHead()
		args, err = lastCommitArgs(s.repository)
		if err != nil {
			return err
		}
		if err := popGitCommand(s.repository, args); err != nil {
			return fmt.Errorf("failed to commit: %v", err)
		}
	case 'm':
		reqReload = true
		// defer s.prompt.writer.HideCursor()
		args := []string{"commit", "--amend", "--quiet"}
		err := popGitCommand(s.repository, args)
		if err != nil {
			return err
		}
		s.repository.LoadHead()
		args, err = lastCommitArgs(s.repository)
		if err != nil {
			return err
		}
		if err := popGitCommand(s.repository, args); err != nil {
			return fmt.Errorf("failed to commit: %v", err)
		}
	case 'a':
		reqReload = true
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = s.repository.Path()
		cmd.Run()
	case 'r':
		reqReload = true
		cmd := exec.Command("git", "reset", "--mixed")
		cmd.Dir = s.repository.Path()
		cmd.Run()
	case '!':
		reqReload = true
		item, err := s.prompt.Selection()
		if err != nil {
			return fmt.Errorf("could not discard changes on item: %v", err)
		}
		entry := item.(*git.StatusEntry)
		args := []string{"checkout", "--", entry.String()}
		cmd := exec.Command("git", args...)
		cmd.Dir = s.repository.Path()
		if err := cmd.Run(); err != nil {
			return err
		}
	case 'q':
		s.prompt.Stop()
	default:
	}
	if reqReload {
		return s.reloadStatus()
	}
	return nil
}

// reloads the list
func (s *status) reloadStatus() error {
	s.repository.LoadHead()
	status, err := s.repository.LoadStatus()
	if err != nil {
		return err
	}
	if len(status.Entities) == 0 {
		// this is the case when the working tree is cleaned at runtime
		s.prompt.Stop()
		s.prompt.SetExitMsg(workingTreeClean(s.repository.Head))
		return nil
	}
	state := s.prompt.State()
	list, err := prompt.NewList(status.Entities, state.ListSize)
	if err != nil {
		return err
	}
	state.List = list
	s.prompt.SetState(state)
	return nil
}

func (s *status) info(item interface{}) [][]term.Cell {
	b := s.repository.Head
	return branchInfo(b, true)
}

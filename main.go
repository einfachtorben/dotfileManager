package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"io/ioutil"
	"log"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

const backupDir = "./dotfiles_backup"

var dryRun bool = false

// checkIfInstalled checks if a package is installed via the package manager
func checkIfInstalled(pkg string) bool {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("command -v %s >/dev/null 2>&1 || dpkg -l %s >/dev/null 2>&1 || pacman -Q %s >/dev/null 2>&1 || rpm -q %s >/dev/null 2>&1 || equery list %s >/dev/null 2>&1", pkg, pkg, pkg, pkg, pkg))
	err := cmd.Run()
	return err == nil
}

// cloneRepo clones a GitHub repo to a temporary directory
func cloneRepo(repoURL, dest string) error {
	_, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL: repoURL,
		Depth: 1,
	})
	return err
}

// backupDotfile backs up an existing dotfile
func backupDotfile(dest string) {
	if _, err := os.Stat(dest); err == nil {
		if dryRun {
			fmt.Printf("[Dry Run] Would back up existing dotfile: %s\n", dest)
			return
		}
		os.MkdirAll(backupDir, os.ModePerm)
		backupPath := filepath.Join(backupDir, filepath.Base(dest))
		os.Rename(dest, backupPath)
		fmt.Printf("Backed up existing dotfile: %s -> %s\n", dest, backupPath)
	}
}

// applyDotfiles applies selected dotfiles by creating symbolic links
func applyDotfiles(repoPath string, selected []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range selected {
		src := filepath.Join(repoPath, file)
		dest := filepath.Join(homeDir, "."+file)
		backupDotfile(dest)
		if dryRun {
			fmt.Printf("[Dry Run] Would create symlink: %s -> %s\n", src, dest)
			continue
		}
		err := os.Symlink(src, dest)
		if err != nil {
			fmt.Printf("Failed to create symlink for %s: %v\n", file, err)
		} else {
			fmt.Printf("Applied dotfile for %s\n", file)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dotfile-manager <GitHub repo URL> [--rollback] [--dry-run]")
		os.Exit(1)
	}

	for _, arg := range os.Args {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	if len(os.Args) == 2 && os.Args[1] == "--rollback" {
		rollbackDotfiles()
		return
	}

	repoURL := os.Args[1]
	tempDir := "./dotfiles_temp"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	fmt.Println("Cloning repository...")
	if err := cloneRepo(repoURL, tempDir); err != nil {
		log.Fatalf("Failed to clone repo: %v", err)
	}

	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		log.Fatal(err)
	}

	availableDotfiles := []string{}
	installedDotfiles := map[string]bool{}

	for _, file := range files {
		if file.IsDir() {
			availableDotfiles = append(availableDotfiles, file.Name())
			installedDotfiles[file.Name()] = checkIfInstalled(file.Name())
		}
	}

	fmt.Println("Launching TUI for selection...")
	selectedDotfiles := tuiSelection(availableDotfiles, installedDotfiles)

	fmt.Println("Applying selected dotfiles...")
	applyDotfiles(tempDir, selectedDotfiles)

	fmt.Println("Dotfiles applied successfully!")
}

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
		os.MkdirAll(backupDir, os.ModePerm)
		backupPath := filepath.Join(backupDir, filepath.Base(dest))
		os.Rename(dest, backupPath)
		fmt.Printf("Backed up existing dotfile: %s -> %s\n", dest, backupPath)
	}
}

// rollbackDotfiles restores backed-up dotfiles
func rollbackDotfiles() {
	files, err := ioutil.ReadDir(backupDir)
	if err != nil {
		log.Fatal("No backups found or failed to read backup directory.")
	}

	homeDir, _ := os.UserHomeDir()
	for _, file := range files {
		backupPath := filepath.Join(backupDir, file.Name())
		originalPath := filepath.Join(homeDir, "."+file.Name())
		os.Rename(backupPath, originalPath)
		fmt.Printf("Restored %s\n", originalPath)
	}
	fmt.Println("Rollback completed.")
}

// tuiSelection provides a UI for selecting dotfiles to apply
func tuiSelection(available []string, installed map[string]bool) []string {
	app := tview.NewApplication()
	list := tview.NewList()

	selected := make(map[string]bool)

	// Populate list with available dotfiles
	for _, item := range available {
		isInstalled := installed[item]
		status := "❌" // Not installed
		if isInstalled {
			status = "✔️"
		}
		selected[item] = isInstalled
		list.AddItem(fmt.Sprintf("[%s] %s", status, item), "Press ENTER to toggle", 0, func() {
			selected[item] = !selected[item]
		})
	}

	// Exit button
	list.AddItem("Apply and Exit", "Proceed with selected dotfiles", 0, func() {
		app.Stop()
	})

	// Run the UI
	if err := app.SetRoot(list, true).Run(); err != nil {
		log.Fatalf("Failed to start TUI: %v", err)
	}

	// Collect selected dotfiles
	var selectedFiles []string
	for file, include := range selected {
		if include {
			selectedFiles = append(selectedFiles, file)
		}
	}

	return selectedFiles
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
		fmt.Println("Usage: dotfile-manager <GitHub repo URL> [--rollback]")
		os.Exit(1)
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

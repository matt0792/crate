package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/matt0792/crate/crate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "projects":
		listProjects()
	case "inspect":
		if len(os.Args) < 3 {
			fmt.Println("Error: project name required")
			fmt.Println("Usage: cratectl inspect <project>")
			os.Exit(1)
		}
		inspectProject(os.Args[2])
	case "drop":
		if len(os.Args) < 3 {
			fmt.Println("Error: project name required")
			fmt.Println("Usage: cratectl drop <project>")
			os.Exit(1)
		}
		dropProject(os.Args[2])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  cratectl projects           List all projects with their sizes")
	fmt.Println("  cratectl inspect <project>  Show namespaces and counts for a project")
	fmt.Println("  cratectl drop <project>     Delete a project (requires sudo)")
}

func getCrateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	return filepath.Join(home, ".crate"), nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func listProjects() {
	crateDir, err := getCrateDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(crateDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No projects found (no .crate directory)")
			return
		}
		fmt.Printf("Error reading crate directory: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tSIZE\t")
	fmt.Fprintln(w, "-------\t----\t")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		walPath := filepath.Join(crateDir, projectName, "crate.wal")

		info, err := os.Stat(walPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(w, "%s\t<no data>\t\n", projectName)
			} else {
				fmt.Fprintf(w, "%s\t<error>\t\n", projectName)
			}
			continue
		}

		fmt.Fprintf(w, "%s\t%s\t\n", projectName, formatSize(info.Size()))
	}

	w.Flush()
}

func inspectProject(projectName string) {
	if err := crate.Project(projectName); err != nil {
		fmt.Printf("Error initializing project: %v\n", err)
		os.Exit(1)
	}

	size, err := crate.Size()
	if err != nil {
		fmt.Printf("Error getting size: %v\n", err)
		os.Exit(1)
	}

	namespaces := crate.Namespaces()

	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("Size: %d bytes\n\n", size)

	if len(namespaces) == 0 {
		fmt.Println("No namespaces found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tCOUNT\t")
	fmt.Fprintln(w, "---------\t-----\t")

	totalCount := 0
	for _, ns := range namespaces {
		count := crate.Count(ns)
		totalCount += count
		fmt.Fprintf(w, "%s\t%d\t\n", ns, count)
	}

	w.Flush()
	fmt.Printf("\nTotal items: %d\n", totalCount)
	fmt.Printf("Total namespaces: %d\n", len(namespaces))
}

func dropProject(projectName string) {
	// Check for root privileges
	if os.Geteuid() != 0 {
		fmt.Println("Error: This command requires sudo privileges")
		fmt.Println("Please run: sudo cratectl drop", projectName)
		os.Exit(1)
	}

	crateDir, err := getCrateDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	projectPath := filepath.Join(crateDir, projectName)

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		fmt.Printf("Error: Project '%s' does not exist\n", projectName)
		os.Exit(1)
	}

	fmt.Printf("WARNING: You are about to permanently delete project '%s'\n", projectName)
	fmt.Print("Type the project name to confirm: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != projectName {
		fmt.Println("Deletion cancelled - project name did not match")
		os.Exit(1)
	}

	if err := os.RemoveAll(projectPath); err != nil {
		fmt.Printf("Error deleting project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted project '%s'\n", projectName)
}

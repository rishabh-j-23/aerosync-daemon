package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Item represents a single selectable option in a menu
type Item struct {
	Label   string
	Handler func() error
}

// Menu represents a collection of items to be displayed via fzf
type Menu struct {
	Title string
	Items []Item
}

// NewMenu initializes a new menu with a title
func NewMenu(title string) *Menu {
	return &Menu{Title: title}
}

// AddItem adds a new selectable item to the menu
func (m *Menu) AddItem(label string, handler func() error) {
	m.Items = append(m.Items, Item{Label: label, Handler: handler})
}

// Display shows the menu once and executes the selected handler
func (m *Menu) Display() (bool, error) {
	var choices []string
	for _, item := range m.Items {
		choices = append(choices, item.Label)
	}

	choice, err := Select(m.Title, choices)
	if err != nil {
		return false, err
	}
	if choice == "" {
		return false, nil // Cancelled
	}

	for _, item := range m.Items {
		if item.Label == choice {
			if item.Handler != nil {
				err := item.Handler()
				return true, err
			}
			return true, nil
		}
	}

	return false, nil
}

// Select is a generic helper to run fzf and get a string selection
func Select(header string, options []string) (string, error) {
	fzfCmd := exec.Command("fzf", "--header", header, "--reverse", "--height", "25%", "--info", "inline")
	fzfCmd.Stdin = strings.NewReader(strings.Join(options, "\n"))
	fzfCmd.Stderr = os.Stderr
	
	out, err := fzfCmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil // User cancelled with ESC
		}
		return "", fmt.Errorf("fzf error: %w", err)
	}
	
	return strings.TrimSpace(string(out)), nil
}

// Prompt is a simple helper to get text input from the user
func Prompt(message string) string {
	fmt.Print(message)
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

// WaitForEnter pauses execution until the user presses enter
func WaitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	fmt.Scanln()
}

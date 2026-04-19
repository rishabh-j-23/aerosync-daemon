package tui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrExit is a sentinel error used to signal that a menu should close
var ErrExit = errors.New("exit menu")

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
		return false, nil // Cancelled with ESC
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

// RunMenu takes a function that returns a Menu, allowing the menu to be rebuilt on each iteration.
// This is useful for dynamic menus where labels or items change based on state.
func RunMenu(builder func() *Menu) error {
	for {
		ClearScreen()
		m := builder()
		selected, err := m.Display()
		if !selected {
			return nil
		}
		if errors.Is(err, ErrExit) {
			return nil
		}
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			WaitForEnter()
		}
	}
}

// Select is a generic helper to run fzf and get a string selection
func Select(header string, options []string) (string, error) {
	// Removed --height 25% to make it fullscreen
	fzfCmd := exec.Command("fzf", "--header", header, "--reverse", "--info", "inline")
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

// ClearScreen clears the terminal screen across different platforms
func ClearScreen() {
	var cmd *exec.Cmd
	if os.PathSeparator == '\\' {
		// Windows
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		// Linux/Unix
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}


// Prompt is a simple helper to get text input from the user, supporting spaces
func Prompt(message string) string {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// WaitForEnter pauses execution until the user presses enter
func WaitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

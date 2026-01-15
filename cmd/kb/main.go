package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pbaille/kb/internal/api"
	"github.com/pbaille/kb/internal/classifier"
	"github.com/pbaille/kb/internal/store"
	"github.com/spf13/cobra"
)

var dbPath string

func main() {
	// Default database location
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".kb", "kb.db")

	rootCmd := &cobra.Command{
		Use:   "kb",
		Short: "Knowledge base with automatic tagging",
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "database path")

	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(showCmd())
	rootCmd.AddCommand(tagsCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getStore() (*store.Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	return store.New(dbPath)
}

func addCmd() *cobra.Command {
	var noClassify bool

	cmd := &cobra.Command{
		Use:   "add [content]",
		Short: "Add a new entry",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")

			s, err := getStore()
			if err != nil {
				return err
			}
			defer s.Close()

			entry, err := s.AddEntry(content)
			if err != nil {
				return err
			}

			fmt.Printf("Added entry: %s\n", entry.ID[:8])
			fmt.Printf("Content: %s\n", truncate(entry.Content, 80))

			// Classification
			if noClassify {
				fmt.Println("(skipped classification)")
				return nil
			}

			clf, err := classifier.New()
			if err != nil {
				fmt.Printf("(classification skipped: %v)\n", err)
				return nil
			}

			// Get existing tags for context
			existingTags, _ := s.ListTags()
			tagNames := make([]string, len(existingTags))
			for i, t := range existingTags {
				tagNames[i] = t.Name
			}

			fmt.Print("Classifying... ")
			result, err := clf.Classify(content, tagNames)
			if err != nil {
				fmt.Printf("failed: %v\n", err)
				return nil
			}

			fmt.Printf("done\n")

			// Create/link tags
			for _, suggestion := range result.Tags {
				var parentID *string

				// Handle parent tag if specified
				if suggestion.Parent != "" {
					parentTag, err := s.GetOrCreateTag(suggestion.Parent, nil)
					if err != nil {
						fmt.Printf("  warning: couldn't create parent tag %s: %v\n", suggestion.Parent, err)
					} else {
						parentID = &parentTag.ID
					}
				}

				tag, err := s.GetOrCreateTag(suggestion.Name, parentID)
				if err != nil {
					fmt.Printf("  warning: couldn't create tag %s: %v\n", suggestion.Name, err)
					continue
				}

				if err := s.LinkEntryTag(entry.ID, tag.ID, suggestion.Confidence); err != nil {
					fmt.Printf("  warning: couldn't link tag %s: %v\n", suggestion.Name, err)
					continue
				}

				if suggestion.Parent != "" {
					fmt.Printf("  + %s (under %s)\n", suggestion.Name, suggestion.Parent)
				} else {
					fmt.Printf("  + %s\n", suggestion.Name)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noClassify, "no-classify", false, "skip automatic classification")
	return cmd
}

func listCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := getStore()
			if err != nil {
				return err
			}
			defer s.Close()

			entries, err := s.ListEntries(limit, 0)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No entries yet. Use 'kb add' to create one.")
				return nil
			}

			for _, e := range entries {
				fmt.Printf("%s  %s\n", e.ID[:8], truncate(e.Content, 60))
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "number of entries to show")
	return cmd
}

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "Show entry details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := getStore()
			if err != nil {
				return err
			}
			defer s.Close()

			// Find entry by prefix
			entries, err := s.ListEntries(100, 0)
			if err != nil {
				return err
			}

			var found *string
			for _, e := range entries {
				if strings.HasPrefix(e.ID, args[0]) {
					found = &e.ID
					break
				}
			}

			if found == nil {
				return fmt.Errorf("entry not found: %s", args[0])
			}

			entry, err := s.GetEntry(*found)
			if err != nil {
				return err
			}

			fmt.Printf("ID:      %s\n", entry.ID)
			fmt.Printf("Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Content:\n%s\n", entry.Content)

			if len(entry.Tags) > 0 {
				fmt.Printf("\nTags:\n")
				for _, t := range entry.Tags {
					fmt.Printf("  - %s\n", t.Name)
				}
			}

			return nil
		},
	}
}

func tagsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "List all tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := getStore()
			if err != nil {
				return err
			}
			defer s.Close()

			tags, err := s.ListTags()
			if err != nil {
				return err
			}

			if len(tags) == 0 {
				fmt.Println("No tags yet. Tags emerge from entry classification.")
				return nil
			}

			// Build hierarchy map
			children := make(map[string][]string)
			roots := []string{}
			tagMap := make(map[string]string) // id -> name

			for _, t := range tags {
				tagMap[t.ID] = t.Name
				if t.ParentID == nil {
					roots = append(roots, t.ID)
				} else {
					children[*t.ParentID] = append(children[*t.ParentID], t.ID)
				}
			}

			// Print tree
			var printTree func(id string, indent int)
			printTree = func(id string, indent int) {
				prefix := strings.Repeat("  ", indent)
				fmt.Printf("%s%s\n", prefix, tagMap[id])
				for _, childID := range children[id] {
					printTree(childID, indent+1)
				}
			}

			for _, rootID := range roots {
				printTree(rootID, 0)
			}

			return nil
		},
	}
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := getStore()
			if err != nil {
				return err
			}
			defer s.Close()

			entries, err := s.SearchEntries(args[0])
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No matching entries found.")
				return nil
			}

			for _, e := range entries {
				fmt.Printf("%s  %s\n", e.ID[:8], truncate(e.Content, 60))
			}

			return nil
		},
	}
}

func truncate(s string, max int) string {
	// Replace newlines with spaces for display
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func serveCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := getStore()
			if err != nil {
				return err
			}
			// Note: don't defer s.Close() as server runs indefinitely

			server := api.New(s, addr)
			return server.Run()
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "server address")
	return cmd
}

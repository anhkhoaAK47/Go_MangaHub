package utils

import (
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"strings"
)

// printMangaTable prints manga in a formatted table
func PrintMangaTable(mangaList []models.Manga) {
	// Column widths
	const (
		idWidth      = 22
		titleWidth   = 25
		authorWidth  = 15
		statusWidth  = 11
		chapterWidth = 10
	)

	divider := fmt.Sprintf("┌%s┬%s┬%s┬%s┬%s┐",
		strings.Repeat("─", idWidth),
		strings.Repeat("─", titleWidth),
		strings.Repeat("─", authorWidth),
		strings.Repeat("─", statusWidth),
		strings.Repeat("─", chapterWidth),
	)
	rowDiv := fmt.Sprintf("├%s┼%s┼%s┼%s┼%s┤",
		strings.Repeat("─", idWidth),
		strings.Repeat("─", titleWidth),
		strings.Repeat("─", authorWidth),
		strings.Repeat("─", statusWidth),
		strings.Repeat("─", chapterWidth),
	)
	bottom := fmt.Sprintf("└%s┴%s┴%s┴%s┴%s┘",
		strings.Repeat("─", idWidth),
		strings.Repeat("─", titleWidth),
		strings.Repeat("─", authorWidth),
		strings.Repeat("─", statusWidth),
		strings.Repeat("─", chapterWidth),
	)

	fmt.Println(divider)
	fmt.Printf("│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│\n",
		idWidth-1, "ID",
		titleWidth-1, "Title",
		authorWidth-1, "Author",
		statusWidth-1, "Status",
		chapterWidth-1, "Chapters",
	)
	fmt.Println(rowDiv)

	for i, m := range mangaList {
		// Truncate long fields to fit columns
		id     := truncate(m.ID, idWidth-2)
		title  := truncate(m.Title, titleWidth-2)
		author := truncate(m.Author, authorWidth-2)
		status := truncate(m.Status, statusWidth-2)
		chapters := fmt.Sprintf("%d", m.TotalChapters)

		fmt.Printf("│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│\n",
			idWidth-1, id,
			titleWidth-1, title,
			authorWidth-1, author,
			statusWidth-1, status,
			chapterWidth-1, chapters,
		)

		if i < len(mangaList)-1 {
			fmt.Println(rowDiv)
		}
	}

	fmt.Println(bottom)
}

// truncate shortens a string to maxLen and adds "..." if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
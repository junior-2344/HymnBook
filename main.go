package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Hymn struct {
	Number string
	Title  string
	Lyrics string
}

func normalizeLanguageName(language string) string {
	normalized := strings.TrimSpace(language)
	if normalized == "" {
		return ""
	}

	aliases := map[string]string{
		"eng":        "English",
		"english":    "English",
		"en":         "English",
		"chichewa":   "Chichewa",
		"chi":        "Chichewa",
		"chewa":      "Chichewa",
		"chitumbuka": "Chitumbuka",
		"tumbuka":    "Chitumbuka",
		"ct":         "Chitumbuka",
	}

	lower := strings.ToLower(normalized)
	if value, ok := aliases[lower]; ok {
		return value
	}

	if match, ok := findLanguageDirectory(normalized); ok {
		return match
	}

	return normalized
}

func findLanguageDirectory(language string) (string, bool) {
	entries, err := os.ReadDir(filepath.Join(".", "hymns"))
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.EqualFold(entry.Name(), language) {
			return entry.Name(), true
		}
	}

	return "", false
}

func discoverLanguages(baseDir string) []string {
	preferred := []string{"English", "Chichewa", "Chitumbuka"}
	found := map[string]bool{}
	results := make([]string, 0, len(preferred))

	entries, err := os.ReadDir(filepath.Join(baseDir, "hymns"))
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				found[strings.TrimSpace(entry.Name())] = true
			}
		}
	}

	for _, language := range preferred {
		if found[language] {
			results = append(results, language)
		}
	}

	for _, language := range preferred {
		if !found[language] {
			if _, ok := found[language]; !ok {
				// no-op, handled by default only when folder exists
			}
		}
	}

	if len(results) > 0 {
		return results
	}

	return preferred
}

func compareHymnNumbers(a, b string) bool {
	first, aErr := strconv.Atoi(a)
	second, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		return first < second
	}
	return a < b
}
func titleFromLyrics(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		} // Only use an explicitly declared title from the lyrics file.
		if strings.HasPrefix(strings.ToLower(trimmed), "title:") {
			return strings.TrimSpace(trimmed[len("title:"):])
		}
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		} // Do NOT assume the first lyric line is the title.
		break
	}
	return ""
}

func prettifyStem(stem string) string {
	stem = strings.ReplaceAll(stem, "_", " ")
	stem = strings.ReplaceAll(stem, "-", " ")
	stem = strings.Join(strings.Fields(stem), " ")
	if stem == "" {
		return "Untitled Hymn"
	}
	return stem
}

func normalizeTitleText(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.ReplaceAll(lower, "&", " and ")
	lower = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ':
			return r
		default:
			return ' '
		}
	}, lower)
	return strings.Join(strings.Fields(lower), " ")
}

func normalizeFileNumber(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "._- ")
	if trimmed == "" {
		return "0"
	}
	if num, err := strconv.Atoi(trimmed); err == nil {
		return strconv.Itoa(num)
	}
	for i, char := range trimmed {
		if char >= '0' && char <= '9' {
			return trimmed[i:]
		}
	}
	return trimmed
}

func parseHymnMetadata(fileName, content string) (string, string) {
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// Preferred format:
	// 1-Holy Holy Holy.txt
	// 2-Amazing Grace.txt
	if idx := strings.LastIndex(stem, "-"); idx >= 0 {
		numberPart := strings.TrimSpace(stem[:idx])
		titlePart := strings.TrimSpace(stem[idx+1:])

		if numberPart != "" && titlePart != "" {
			return normalizeFileNumber(numberPart), titlePart
		}
	}
	// If there is no title in the filename, only use an
	// explicitly declared Title: or # title from the file.
	if title := titleFromLyrics(content); title != "" {
		return normalizeFileNumber(stem), title
	}
	// Last resort: use the filename itself.
	return normalizeFileNumber(stem), prettifyStem(stem)

}

func filterHymns(hymns []Hymn, query string) []Hymn {
	searchTerm := strings.ToLower(strings.TrimSpace(query))
	if searchTerm == "" {
		filtered := make([]Hymn, len(hymns))
		copy(filtered, hymns)
		return filtered
	}

	filtered := make([]Hymn, 0, len(hymns))
	for _, hymn := range hymns {
		haystack := strings.ToLower(fmt.Sprintf("%s %s", hymn.Number, hymn.Title))
		if strings.Contains(haystack, searchTerm) {
			filtered = append(filtered, hymn)
		}
	}
	return filtered
}

func stripDuplicateTitle(content, title string) string {
	lines := strings.Split(
		strings.ReplaceAll(content, "\r\n", "\n"),
		"\n",
	)

	normalizedTitle := normalizeTitleText(title)

	if normalizedTitle == "" {
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	// Find the first non-empty line.
	firstContentLine := -1

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentLine = i
			break
		}
	}

	// If the first lyric line is exactly the same as the title,
	// remove it. Do not remove later occurrences because they
	// may be legitimate lyrics.
	if firstContentLine >= 0 {
		firstLine := strings.TrimSpace(lines[firstContentLine])

		if normalizeTitleText(firstLine) == normalizedTitle {
			lines = append(
				lines[:firstContentLine],
				lines[firstContentLine+1:]...,
			)
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func parseLyricsIntoStanzas(content string) []string {
	// Normalize all line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")

	stanzas := make([]string, 0)
	currentStanza := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// A completely blank line means:
		// "the current stanza has ended".
		if trimmed == "" {
			if len(currentStanza) > 0 {
				stanzas = append(
					stanzas,
					strings.Join(currentStanza, "\n"),
				)

				currentStanza = make([]string, 0)
			}

			continue
		}

		// Keep the original lyric line.
		currentStanza = append(currentStanza, trimmed)
	}

	// Add the final stanza if the file does not
	// end with a blank line.
	if len(currentStanza) > 0 {
		stanzas = append(
			stanzas,
			strings.Join(currentStanza, "\n"),
		)
	}

	return stanzas
}

func loadHymnsForLanguage(baseDir, language string) ([]Hymn, error) {
	resolvedLanguage := normalizeLanguageName(language)
	if resolvedLanguage == "" {
		return nil, fmt.Errorf("language is required")
	}

	directoryPath := filepath.Join(baseDir, "hymns", resolvedLanguage)
	if _, err := os.Stat(directoryPath); err != nil {
		if match, ok := findLanguageDirectory(resolvedLanguage); ok {
			directoryPath = filepath.Join(baseDir, "hymns", match)
		} else {
			return nil, err
		}
	}

	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}

	hymns := make([]Hymn, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".txt" && ext != ".md" && ext != ".text" {
			continue
		}

		filePath := filepath.Join(directoryPath, entry.Name())
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			continue
		}

		number, title := parseHymnMetadata(entry.Name(), string(content))
		hymns = append(hymns, Hymn{
			Number: number,
			Title:  title,
			Lyrics: string(content),
		})
	}

	sort.Slice(hymns, func(i, j int) bool {
		return compareHymnNumbers(hymns[i].Number, hymns[j].Number)
	})

	return hymns, nil
}

func loadLyrics(language, number string) string {
	language = normalizeLanguageName(language)
	if language == "" {
		return "*Lyrics file not found for this song yet.*"
	}

	hymns, err := loadHymnsForLanguage(".", language)
	if err != nil {
		return "*Lyrics file not found for this song yet.*"
	}

	for _, hymn := range hymns {
		if strings.EqualFold(hymn.Number, number) {
			return hymn.Lyrics
		}
	}

	return "*Lyrics file not found for this song yet.*"
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Church Hymnal")

	availableLanguages := discoverLanguages(".")
	currentLanguage := "English"
	if len(availableLanguages) > 0 {
		currentLanguage = normalizeLanguageName(availableLanguages[0])
	}

	var currentHymns []Hymn
	var filteredHymns []Hymn
	var hymnList *widget.List

	searchBar := widget.NewEntry()
	// Main lyrics container.
	// We will put the title, separator and individual stanzas inside it.
	lyricContent := container.NewVBox()

	lyricScroll := container.NewVScroll(
		container.NewPadded(lyricContent),
	)

	hymnList = widget.NewList(
		func() int { return len(filteredHymns) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(filteredHymns) {
				return
			}
			hymn := filteredHymns[id]
			item.(*widget.Label).SetText(fmt.Sprintf("%s. %s", hymn.Number, hymn.Title))
		},
	)

	hymnList.OnSelected = func(id widget.ListItemID) {
		if id >= len(filteredHymns) {
			return
		}

		selected := filteredHymns[id]

		// Read the original lyrics.
		lyricsContent := loadLyrics(
			currentLanguage,
			selected.Number,
		)

		// Remove the duplicated title only if it is
		// the first actual lyric line.
		lyricsContent = stripDuplicateTitle(
			lyricsContent,
			selected.Title,
		)

		// Convert the file into actual stanzas.
		stanzas := parseLyricsIntoStanzas(lyricsContent)

		// Create a completely new list of UI objects.
		objects := make([]fyne.CanvasObject, 0)

		// ------------------------------------------------
		// TITLE
		// ------------------------------------------------

		titleLabel := widget.NewLabel(
			fmt.Sprintf("%s. %s", selected.Number, selected.Title),
		)

		titleLabel.TextStyle = fyne.TextStyle{
			Bold: true,
		}

		titleLabel.Alignment = fyne.TextAlignCenter

		objects = append(objects, titleLabel)

		// ------------------------------------------------
		// SEPARATOR
		// ------------------------------------------------

		objects = append(
			objects,
			widget.NewSeparator(),
		)

		// ------------------------------------------------
		// STANZAS
		// ------------------------------------------------

		for _, stanza := range stanzas {

			// Each stanza is ONE Label.
			//
			// Newlines inside the label are preserved,
			// so all lines belonging to this stanza stay
			// together.
			stanzaLabel := widget.NewLabel(stanza)

			// Allow the text to wrap if the window is narrow.
			stanzaLabel.Wrapping = fyne.TextWrapWord

			// Left-align the lyrics.
			stanzaLabel.Alignment = fyne.TextAlignLeading

			objects = append(
				objects,
				stanzaLabel,
			)

			// Add actual vertical space AFTER every stanza.
			//
			// This is NOT a Unicode space.
			// It is a real Fyne layout spacer.
			spacer := canvas.NewRectangle(
				color.Transparent,
			)

			spacer.SetMinSize(
				fyne.NewSize(1, 20),
			)

			objects = append(
				objects,
				spacer,
			)
		}

		// Replace the old content with the new hymn.
		lyricContent.Objects = objects

		// Tell Fyne that the container changed.
		lyricContent.Refresh()

		// Return to the beginning of the hymn.
		lyricScroll.ScrollToTop()
	}

	applyFilters := func() {
		filteredHymns = filterHymns(currentHymns, searchBar.Text)
		hymnList.Refresh()
	}

	refreshLanguageHymns := func() {
		loadedHymns, err := loadHymnsForLanguage(".", currentLanguage)
		if err != nil || len(loadedHymns) == 0 {
			currentHymns = nil
			filteredHymns = nil
			lyricContent.RemoveAll()
			lyricContent.Add(widget.NewLabel("### No hymns found for this language yet."))
			hymnList.Refresh()
			return
		}
		currentHymns = loadedHymns
		applyFilters()
	}

	searchBar.SetPlaceHolder("Search this language...")
	searchBar.OnChanged = func(text string) {
		applyFilters()
	}

	langDropdown := widget.NewSelect(availableLanguages, func(value string) {
		currentLanguage = normalizeLanguageName(value)
		searchBar.SetText("")
		refreshLanguageHymns()
	})
	langDropdown.SetSelected(currentLanguage)

	refreshLanguageHymns()

	sidebarControls := container.NewVBox(langDropdown, searchBar)
	sidebarLayout := container.NewBorder(sidebarControls, nil, nil, nil, hymnList)

	mainSplit := container.NewHSplit(sidebarLayout, lyricScroll)
	mainSplit.Offset = 0.35

	myWindow.SetContent(mainSplit)
	myWindow.Resize(fyne.NewSize(850, 600))
	myWindow.ShowAndRun()
}

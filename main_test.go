package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHymnsForLanguageLoadsSelectedLanguageOnly(t *testing.T) {
	tempDir := t.TempDir()
	hymnsDir := filepath.Join(tempDir, "hymns")
	if err := os.MkdirAll(filepath.Join(hymnsDir, "English"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hymnsDir, "Chichewa"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(hymnsDir, "English", "1.txt"), []byte("Amazing Grace\nLyrics A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hymnsDir, "English", "2.txt"), []byte("How Great Thou Art\nLyrics B"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hymnsDir, "Chichewa", "1.txt"), []byte("Moyo Wanga\nChichewa lyrics"), 0o644); err != nil {
		t.Fatal(err)
	}

	hymns, err := loadHymnsForLanguage(tempDir, "English")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hymns) != 2 {
		t.Fatalf("expected 2 English hymns, got %d", len(hymns))
	}
	if hymns[0].Title != "Amazing Grace" || hymns[1].Title != "How Great Thou Art" {
		t.Fatalf("unexpected English hymns loaded: %#v", hymns)
	}
}

func TestFilterHymnsRespectsSelectedLanguageData(t *testing.T) {
	hymns := []Hymn{
		{Number: "1", Title: "Amazing Grace"},
		{Number: "2", Title: "How Great Thou Art"},
		{Number: "3", Title: "Moyo Wanga"},
	}

	filtered := filterHymns(hymns, "grace")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 matching hymn, got %d", len(filtered))
	}
	if filtered[0].Title != "Amazing Grace" {
		t.Fatalf("expected Amazing Grace to be returned, got %q", filtered[0].Title)
	}
}

func TestNormalizeLanguageName(t *testing.T) {
	if got := normalizeLanguageName("eng"); got != "English" {
		t.Fatalf("expected eng to normalize to English, got %q", got)
	}
	if got := normalizeLanguageName("english"); got != "English" {
		t.Fatalf("expected english to normalize to English, got %q", got)
	}
	if got := normalizeLanguageName("Chichewa"); got != "Chichewa" {
		t.Fatalf("expected Chichewa to stay Chichewa, got %q", got)
	}
	for _, alias := range []string{"Chitumbuka", "tumbuka", "ct"} {
		if got := normalizeLanguageName(alias); got != "tumbuka" {
			t.Errorf("expected %q to normalize to tumbuka, got %q", alias, got)
		}
	}
}

func TestChitumbukaLoadsFromTumbukaDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tumbukaDir := filepath.Join(tempDir, "hymns", "tumbuka")
	if err := os.MkdirAll(tumbukaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tumbukaDir, "1-Yesu ndi Mponoski.md"), []byte("Lyrics"), 0o644); err != nil {
		t.Fatal(err)
	}

	languages := discoverLanguages(tempDir)
	if len(languages) != 1 || languages[0] != "Chitumbuka" {
		t.Fatalf("expected the tumbuka directory to be listed as Chitumbuka, got %#v", languages)
	}

	hymns, err := loadHymnsForLanguage(tempDir, languages[0])
	if err != nil {
		t.Fatalf("expected Chitumbuka hymns to load from tumbuka, got %v", err)
	}
	if len(hymns) != 1 || hymns[0].Title != "Yesu ndi Mponoski" {
		t.Fatalf("unexpected hymns loaded from tumbuka: %#v", hymns)
	}
}

func TestStripDuplicateTitleAndFormatLyricsForMarkdown(t *testing.T) {
	content := "Amazing Grace\nAmazing grace, how sweet the sound\nThat saved a wretch like me\n"
	clean := stripDuplicateTitle(content, "Amazing Grace")
	if clean == "Amazing Grace" {
		t.Fatalf("expected duplicate title to be removed, got %q", clean)
	}
	if !strings.Contains(clean, "Amazing grace, how sweet the sound") {
		t.Fatalf("expected lyric line to remain, got %q", clean)
	}

	formatted := formatLyricsForMarkdown(clean)
	if !strings.Contains(formatted, "  \n") {
		t.Fatalf("expected markdown hardbreaks, got %q", formatted)
	}
}

func TestDuplicateTitleIsRemovedWhenFirstLyricMatchesFilename(t *testing.T) {
	content := "Holy, Holy, Holy,\nLord God Almighty!\nEarly in the morning\n"
	clean := stripDuplicateTitle(content, "Holy Holy")
	if strings.Contains(strings.ToLower(clean), "holy, holy, holy") {
		t.Fatalf("expected duplicated title line removed, got %q", clean)
	}
	if !strings.Contains(clean, "Lord God Almighty!") {
		t.Fatalf("expected lyric text to remain, got %q", clean)
	}
}

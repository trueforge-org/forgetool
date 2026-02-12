package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceInFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		pattern     string
		replacement string
		want        string
		wantErr     bool
	}{
		{
			name:        "Simple replacement",
			content:     "Hello world",
			pattern:     "world",
			replacement: "universe",
			want:        "Hello universe",
			wantErr:     false,
		},
		{
			name:        "Multiple occurrences",
			content:     "foo bar foo baz foo",
			pattern:     "foo",
			replacement: "qux",
			want:        "qux bar qux baz qux",
			wantErr:     false,
		},
		{
			name:        "No match",
			content:     "Hello world",
			pattern:     "xyz",
			replacement: "abc",
			want:        "Hello world",
			wantErr:     false,
		},
		{
			name:        "Empty pattern - no replacement when pattern is empty",
			content:     "Hello world",
			pattern:     "",
			replacement: "X",
			want:        "XHXeXlXlXoX XwXoXrXlXdX", // strings.ReplaceAll with empty string replaces between chars
			wantErr:     false,
		},
		{
			name:        "Empty replacement",
			content:     "Hello world",
			pattern:     "world",
			replacement: "",
			want:        "Hello ",
			wantErr:     false,
		},
		{
			name:        "Multiline content",
			content:     "line1\nline2\nline3",
			pattern:     "line2",
			replacement: "REPLACED",
			want:        "line1\nREPLACED\nline3",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "test.txt")

			// Write initial content
			if err := os.WriteFile(filename, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Perform replacement
			err := ReplaceInFile(filename, tt.pattern, tt.replacement)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceInFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Read the modified content
				got, err := os.ReadFile(filename)
				if err != nil {
					t.Fatal(err)
				}

				if string(got) != tt.want {
					t.Errorf("ReplaceInFile() = %q, want %q", string(got), tt.want)
				}
			}
		})
	}
}

func TestReplaceInFile_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "nonexistent.txt")

	err := ReplaceInFile(filename, "pattern", "replacement")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestReplaceContentBetweenLines(t *testing.T) {
	tests := []struct {
		name          string
		targetContent string
		sourceContent string
		from          string
		till          string
		want          string
		wantErr       bool
	}{
		{
			name: "Replace content between markers",
			targetContent: `line1
<!-- START -->
old content
<!-- END -->
line2`,
			sourceContent: `<!-- START -->
new content
<!-- END -->`,
			from: "<!-- START -->",
			till: "<!-- END -->",
			want: `line1
<!-- START -->

new content

<!-- END -->
line2`,
			wantErr: false,
		},
		{
			name: "Replace with empty source",
			targetContent: `line1
<!-- START -->
old content
<!-- END -->
line2`,
			sourceContent: `<!-- START -->
<!-- END -->`,
			from: "<!-- START -->",
			till: "<!-- END -->",
			want: `line1
<!-- START -->


<!-- END -->
line2`,
			wantErr: false,
		},
		{
			name: "Multiple lines replaced",
			targetContent: `header
# BEGIN
line1
line2
line3
# END
footer`,
			sourceContent: `# BEGIN
replacement
# END`,
			from: "# BEGIN",
			till: "# END",
			want: `header
# BEGIN

replacement

# END
footer`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetFile := filepath.Join(tmpDir, "target.txt")
			sourceFile := filepath.Join(tmpDir, "source.txt")

			// Write files
			if err := os.WriteFile(targetFile, []byte(tt.targetContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(sourceFile, []byte(tt.sourceContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Perform replacement
			err := ReplaceContentBetweenLines(targetFile, sourceFile, tt.from, tt.till)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceContentBetweenLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Read the modified content
				got, err := os.ReadFile(targetFile)
				if err != nil {
					t.Fatal(err)
				}

				if string(got) != tt.want {
					t.Errorf("ReplaceContentBetweenLines() =\n%q\nwant\n%q", string(got), tt.want)
				}
			}
		})
	}
}

func TestReplaceContentBetweenLines_NonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	sourceFile := filepath.Join(tmpDir, "nonexistent.txt")

	if err := os.WriteFile(targetFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ReplaceContentBetweenLines(targetFile, sourceFile, "START", "END")
	if err == nil {
		t.Error("Expected error for non-existent source file, got nil")
	}
}

func TestReplaceContentBetweenLines_NonExistentTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "nonexistent.txt")
	sourceFile := filepath.Join(tmpDir, "source.txt")

	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ReplaceContentBetweenLines(targetFile, sourceFile, "START", "END")
	if err == nil {
		t.Error("Expected error for non-existent target file, got nil")
	}
}

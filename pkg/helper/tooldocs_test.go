package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeterminePaths(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		wantSubDir     string
		wantNewFileName string
	}{
		{
			name:            "Simple command file",
			filename:        "adv_testcmd.md",
			wantSubDir:      "adv",
			wantNewFileName: "testcmd.md",
		},
		{
			name:            "Index file - matching subdirectory and filename",
			filename:        "cluster_cluster.md",
			wantSubDir:      "cluster",
			wantNewFileName: "index.md",
		},
		{
			name:            "No underscore in filename",
			filename:        "forgetool.md",
			wantSubDir:      "",
			wantNewFileName: "forgetool.md",
		},
		{
			name:            "Multiple underscores",
			filename:        "charts_bump_version.md",
			wantSubDir:      "charts",
			wantNewFileName: "bump_version.md",
		},
		{
			name:            "Talos command",
			filename:        "talos_apply.md",
			wantSubDir:      "talos",
			wantNewFileName: "apply.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSubDir, gotNewFileName := determinePaths(tt.filename)
			if gotSubDir != tt.wantSubDir {
				t.Errorf("determinePaths() subDir = %v, want %v", gotSubDir, tt.wantSubDir)
			}
			if gotNewFileName != tt.wantNewFileName {
				t.Errorf("determinePaths() newFileName = %v, want %v", gotNewFileName, tt.wantNewFileName)
			}
		})
	}
}

func TestAddYamlTitle(t *testing.T) {
	tests := []struct {
		name            string
		content         []byte
		isPrimaryIndex  bool
		wantTitle       string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:           "Primary index",
			content:        []byte("## forgetool\n\nSome content\n"),
			isPrimaryIndex: true,
			wantTitle:      "commands",
			wantContains:   []string{"---", "title: commands", "---"},
		},
		{
			name:           "Regular command with SEE ALSO",
			content:        []byte("## forgetool apply\n\nApply command\n\n### SEE ALSO\n\n* parent command\n"),
			isPrimaryIndex: false,
			wantTitle:      "apply",
			wantContains:   []string{"title: apply", "Apply command"},
			wantNotContains: []string{"SEE ALSO", "parent command"},
		},
		{
			name:           "Command with forgetool prefix",
			content:        []byte("## forgetool cluster init\n\nInitialize cluster\n"),
			isPrimaryIndex: false,
			wantTitle:      "cluster init",
			wantContains:   []string{"title: cluster init", "Initialize cluster"},
		},
		{
			name:           "Simple forgetool command",
			content:        []byte("## forgetool\n\nMain command\n"),
			isPrimaryIndex: false,
			wantTitle:      "command",
			wantContains:   []string{"title: command", "Main command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addYamlTitle(tt.content, tt.isPrimaryIndex)
			gotStr := string(got)

			for _, want := range tt.wantContains {
				if !strings.Contains(gotStr, want) {
					t.Errorf("addYamlTitle() output should contain %q, got:\n%s", want, gotStr)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(gotStr, notWant) {
					t.Errorf("addYamlTitle() output should not contain %q, got:\n%s", notWant, gotStr)
				}
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (path string, content []byte, fileInfo os.DirEntry)
		wantErr bool
	}{
		{
			name: "Write file successfully",
			setup: func(t *testing.T) (string, []byte, os.DirEntry) {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "subdir", "test.md")
				content := []byte("# Test Content\n")
				
				// Create a dummy file to get DirEntry
				dummyPath := filepath.Join(tmpDir, "dummy.txt")
				os.WriteFile(dummyPath, []byte("test"), 0644)
				entries, _ := os.ReadDir(tmpDir)
				
				return path, content, entries[0]
			},
			wantErr: false,
		},
		{
			name: "Create nested directories",
			setup: func(t *testing.T) (string, []byte, os.DirEntry) {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "level1", "level2", "level3", "file.md")
				content := []byte("deep content")
				
				dummyPath := filepath.Join(tmpDir, "dummy.txt")
				os.WriteFile(dummyPath, []byte("test"), 0644)
				entries, _ := os.ReadDir(tmpDir)
				
				return path, content, entries[0]
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, content, fileInfo := tt.setup(t)

			err := writeToFile(path, content, fileInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("File should have been created")
				}

				// Verify content
				gotContent, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read created file: %v", err)
				}
				if string(gotContent) != string(content) {
					t.Errorf("File content = %q, want %q", gotContent, content)
				}
			}
		})
	}
}

func TestRenameForgetoolToIndex(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "Rename forgetool.md to index.md",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				forgetoolPath := filepath.Join(tmpDir, "forgetool.md")
				os.WriteFile(forgetoolPath, []byte("# Forgetool"), 0644)
				return tmpDir
			},
			wantErr: false,
		},
		{
			name: "No forgetool.md file - no error",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)

			err := renameForgetoolToIndex(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("renameForgetoolToIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.name == "Rename forgetool.md to index.md" {
				indexPath := filepath.Join(dir, "index.md")
				if _, err := os.Stat(indexPath); os.IsNotExist(err) {
					t.Error("index.md should have been created")
				}

				forgetoolPath := filepath.Join(dir, "forgetool.md")
				if _, err := os.Stat(forgetoolPath); !os.IsNotExist(err) {
					t.Error("forgetool.md should have been renamed")
				}
			}
		})
	}
}

// Note: ToolDocs, processFiles, and moveMatchingFilesToSubdirs are integration
// functions that coordinate multiple file operations. They are better tested
// through integration tests rather than unit tests, as they require complex
// directory structures and file setups.

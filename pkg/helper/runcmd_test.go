package helper

import (
	"bytes"
	"strings"
	"testing"
)

func TestFilteredWriter_Write(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		filters []string
		want    string
	}{
		{
			name:    "No filters",
			input:   "line1\nline2\nline3",
			filters: []string{},
			want:    "line1\nline2\nline3",
		},
		{
			name:    "Filter one line",
			input:   "line1\nerror: filtered\nline3",
			filters: []string{"filtered"},
			want:    "line1\nline3",
		},
		{
			name:    "Filter multiple lines",
			input:   "line1\ncertificate signed by unknown authority\nline3\nbootstrap is not available yet\nline5",
			filters: []string{"certificate signed by unknown authority", "bootstrap is not available yet"},
			want:    "line1\nline3\nline5",
		},
		{
			name:    "No matches",
			input:   "line1\nline2\nline3",
			filters: []string{"notfound"},
			want:    "line1\nline2\nline3",
		},
		{
			name:    "Empty input",
			input:   "",
			filters: []string{"filter"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			fw := &filteredWriter{
				writer:  &buf,
				filters: tt.filters,
			}

			n, err := fw.Write([]byte(tt.input))
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Write returns the number of bytes written to the underlying writer
			// which may be less than input if lines are filtered
			_ = n

			got := buf.String()
			if got != tt.want {
				t.Errorf("Write() output:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     []string
		silent      bool
		wantErr     bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:    "Echo command silent",
			command: []string{"echo", "hello world"},
			silent:  true,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "hello world") {
					t.Errorf("Expected output to contain 'hello world', got: %q", output)
				}
			},
		},
		{
			name:    "Echo command not silent",
			command: []string{"echo", "test output"},
			silent:  false,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "test output") {
					t.Errorf("Expected output to contain 'test output', got: %q", output)
				}
			},
		},
		{
			name:    "Command with multiple arguments",
			command: []string{"echo", "arg1", "arg2", "arg3"},
			silent:  true,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "arg1 arg2 arg3") {
					t.Errorf("Expected output to contain arguments, got: %q", output)
				}
			},
		},
		{
			name:    "Nonexistent command",
			command: []string{"nonexistent_command_12345"},
			silent:  true,
			wantErr: true,
			checkOutput: func(t *testing.T, output string) {
				// Error case, output may be empty
			},
		},
		{
			name:    "Command that fails",
			command: []string{"sh", "-c", "exit 1"},
			silent:  true,
			wantErr: true,
			checkOutput: func(t *testing.T, output string) {
				// Error case
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RunCommand(tt.command, tt.silent)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	// Test with empty command slice - should panic or error
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with empty command slice")
		}
	}()

	_, _ = RunCommand([]string{}, true)
}

func TestRunCommand_FilteredOutput(t *testing.T) {
	// Test that filtered strings don't appear in non-silent output
	// This is harder to test without capturing stdout/stderr
	// For now, we just verify the command runs
	command := []string{"echo", "certificate signed by unknown authority"}
	output, err := RunCommand(command, true)
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}

	// In silent mode, output should still contain the text
	if !strings.Contains(output, "certificate signed by unknown authority") {
		t.Error("Expected filtered text in silent mode output")
	}
}

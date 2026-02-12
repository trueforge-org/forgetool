package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/trueforge-org/forgetool/pkg/helper"
)

var (
	genToolDocsMkdirAll = os.MkdirAll
	genToolDocsMarkdown = doc.GenMarkdownTree
	genToolDocsWriter   = helper.ToolDocs
)

func runGenToolDocs(outdir string) error {
	tmpdir := helper.DocsCache

	if err := genToolDocsMkdirAll(tmpdir, 0o777); err != nil {
		return fmt.Errorf("create docs cache: %w", err)
	}

	if err := genToolDocsMarkdown(RootCmd, tmpdir); err != nil {
		return fmt.Errorf("generate markdown tree: %w", err)
	}
	genToolDocsWriter(tmpdir, outdir)
	return nil
}

var gendocsCmd = &cobra.Command{
	Use:    "gentooldocs",
	Short:  "Generate documentation for the CLI",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGenToolDocs(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	adv.AddCommand(gendocsCmd)
}

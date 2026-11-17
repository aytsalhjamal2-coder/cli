package ai

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/search/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/search"
	"github.com/spf13/cobra"
)

// AIOptions configures searching for open-source AI repositories.
type AIOptions struct {
	Exporter cmdutil.Exporter
	IO       *iostreams.IOStreams
	Query    search.Query
	Searcher search.Searcher
}

// NewCmdAI creates the "search ai" command.
func NewCmdAI(f *cmdutil.Factory, runF func(*AIOptions) error) *cobra.Command {
	var license string
	opts := &AIOptions{
		IO: f.IOStreams,
		Query: search.Query{
			Kind:  search.KindRepositories,
			Limit: 30,
			Order: "desc",
			Sort:  "stars",
		},
	}

	cmd := &cobra.Command{
		Use:   "ai [<query>...]",
		Short: "Search for open-source AI projects and models",
		Long: heredoc.Doc(`
			Search GitHub repositories for open-source AI projects and models.
			By default, results are limited to repositories with an MIT license and
			sorted by stars. Use --license to search for another GitHub license key.
		`),
		Example: heredoc.Doc(`
			# Find popular MIT-licensed AI repositories
			$ gh search ai

			# Find Apache-2.0-licensed language models
			$ gh search ai "language model" --license apache-2.0
		`),
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if opts.Query.Limit < 1 || opts.Query.Limit > shared.SearchMaxResults {
				return cmdutil.FlagErrorf("`--limit` must be between 1 and 1000")
			}
			if license == "" {
				return cmdutil.FlagErrorf("`--license` must not be empty")
			}
			if len(args) == 0 {
				opts.Query.Keywords = []string{"ai"}
			} else {
				opts.Query.Keywords = args
			}
			opts.Query.Qualifiers.License = []string{license}
			if runF != nil {
				return runF(opts)
			}
			var err error
			opts.Searcher, err = shared.Searcher(f)
			if err != nil {
				return err
			}
			return aiRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.Exporter, search.RepositoryFields)
	cmd.Flags().IntVarP(&opts.Query.Limit, "limit", "L", 30, "Maximum number of repositories to fetch")
	cmd.Flags().StringVar(&license, "license", "mit", "Filter by GitHub license key")

	return cmd
}

func aiRun(opts *AIOptions) error {
	io := opts.IO
	io.StartProgressIndicator()
	result, err := opts.Searcher.Repositories(opts.Query)
	io.StopProgressIndicator()
	if err != nil {
		return err
	}
	if len(result.Items) == 0 && opts.Exporter == nil {
		return cmdutil.NewNoResultsError("no open-source AI repositories matched your search")
	}
	if err := io.StartPager(); err == nil {
		defer io.StopPager()
	} else {
		fmt.Fprintf(io.ErrOut, "failed to start pager: %v\n", err)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(io, result.Items)
	}

	return displayResults(io, result)
}

func displayResults(io *iostreams.IOStreams, results search.RepositoriesResult) error {
	cs := io.ColorScheme()
	tp := tableprinter.New(io, tableprinter.WithHeader("Repository", "Description", "Language", "License", "Stars"))
	for _, repo := range results.Items {
		tp.AddField(repo.FullName, tableprinter.WithColor(cs.Bold))
		tp.AddField(text.RemoveExcessiveWhitespace(repo.Description))
		tp.AddField(repo.Language)
		tp.AddField(repo.License.Name)
		tp.AddField(strconv.Itoa(repo.StargazersCount))
		tp.EndRow()
	}
	if io.IsStdoutTTY() {
		fmt.Fprintf(io.Out, "\nShowing %d of %d repositories\n\n", len(results.Items), results.Total)
	}
	return tp.Render()
}

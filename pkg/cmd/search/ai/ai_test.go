package ai

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/search"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdAI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  search.Query
	}{
		{
			name: "defaults",
			want: search.Query{
				Keywords:   []string{"ai"},
				Kind:       search.KindRepositories,
				Limit:      30,
				Order:      "desc",
				Sort:       "stars",
				Qualifiers: search.Qualifiers{License: []string{"mit"}},
			},
		},
		{
			name:  "query, license, and limit",
			input: `"language model" --license apache-2.0 --limit 12`,
			want: search.Query{
				Keywords:   []string{"language model"},
				Kind:       search.KindRepositories,
				Limit:      12,
				Order:      "desc",
				Sort:       "stars",
				Qualifiers: search.Qualifiers{License: []string{"apache-2.0"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{IOStreams: ios}
			argv, err := shlex.Split(tt.input)
			require.NoError(t, err)
			var got *AIOptions
			cmd := NewCmdAI(f, func(opts *AIOptions) error {
				got = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.Query)
		})
	}
}

func TestNewCmdAIValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "invalid limit", args: []string{"ai", "--limit", "1001"}, wantErr: "`--limit` must be between 1 and 1000"},
		{name: "empty license", args: []string{"ai", "--license", ""}, wantErr: "`--license` must not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			cmd := NewCmdAI(&cmdutil.Factory{IOStreams: ios}, func(*AIOptions) error {
				t.Fatal("run function should not be called for invalid arguments")
				return nil
			})
			cmd.SetArgs(tt.args[1:])
			cmd.SetIn(&bytes.Buffer{})

			_, err := cmd.ExecuteC()
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestAIRun(t *testing.T) {
	tests := []struct {
		name       string
		result     search.RepositoriesResult
		searchErr  error
		wantErr    string
		wantStdout string
	}{
		{
			name: "lists useful repository fields",
			result: search.RepositoriesResult{
				Total: 40,
				Items: []search.Repository{{
					FullName:        "org/model",
					Description:     "An   open-source model",
					Language:        "Python",
					License:         search.License{Name: "MIT License"},
					StargazersCount: 123,
				}},
			},
			wantStdout: "org/model\tAn open-source model\tPython\tMIT License\t123\n",
		},
		{
			name:    "no results",
			wantErr: "no open-source AI repositories matched your search",
		},
		{
			name:      "search error",
			searchErr: errors.New("GitHub search failed"),
			wantErr:   "GitHub search failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			opts := &AIOptions{
				IO: ios,
				Query: search.Query{
					Keywords: []string{"ai"},
					Kind:     search.KindRepositories,
					Limit:    30,
					Order:    "desc",
					Sort:     "stars",
					Qualifiers: search.Qualifiers{
						License: []string{"mit"},
					},
				},
			}
			opts.Searcher = &search.SearcherMock{
				RepositoriesFunc: func(query search.Query) (search.RepositoriesResult, error) {
					assert.Equal(t, opts.Query, query)
					return tt.result, tt.searchErr
				},
			}

			err := aiRun(opts)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Empty(t, stderr.String())
		})
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/face-match/internal/app"
	"github.com/face-match/internal/service"
	"github.com/face-match/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Config *app.Config
	Pool   *pgxpool.Pool
}

func main() {
	config := &app.Config{
		AIEndpoint:  os.Getenv("AI_ENDPOINT"),
		DatabaseUrl: os.Getenv("DATABASE_URL"),
		DataRoot:    os.Getenv("DATA_ROOT"),
	}
	config.InputPath = filepath.Join(config.DataRoot, "/ingest/input")
	config.FinishedPath = filepath.Join(config.DataRoot, "/ingest/finished")
	config.ThumbsPath = filepath.Join(config.DataRoot, "/images/thumbs")

	dependencies := &Dependencies{
		Config: config,
	}

	var rootCmd = &cobra.Command{
		Use:   "ingest",
		Short: "Ingestion tool for the face match website.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if dependencies.Pool == nil {
				pool, err := store.Open(cmd.Context(), config.DatabaseUrl)
				if err != nil {
					return err
				}
				dependencies.Pool = pool
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if dependencies.Pool != nil {
				dependencies.Pool.Close()
				dependencies.Pool = nil
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&config.DatabaseUrl, "database-url", config.DatabaseUrl, "Database URL")
	rootCmd.PersistentFlags().StringVar(&config.DataRoot, "data-root", config.DataRoot, "Data root directory")

	rootCmd.AddCommand(cmdCategories(dependencies))
	rootCmd.AddCommand(cmdImport(dependencies))
	rootCmd.AddCommand(cmdSearch(dependencies))
	rootCmd.AddCommand(cmdPerson(dependencies))

	ctx := context.Background()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}

func cmdCategories(dependencies *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List all categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := store.NewCategoryStore(dependencies.Pool)
			categories, err := cs.List(cmd.Context())
			if err != nil {
				return err
			}
			for _, c := range categories {
				fmt.Printf("%d\t%s\tnsfw=%v\n", c.ID, c.DisplayName, c.IsNsfw)
			}
			return nil
		},
	}
}

func cmdImport(dependencies *Dependencies) *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import all images in the ingest input folder for a single category.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := service.NewImportService(dependencies.Config, dependencies.Pool)
			if err := s.Import(cmd.Context(), category); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Category (required; applies to all input files)")
	_ = cmd.MarkFlagRequired("category")

	return cmd
}

func cmdSearch(dependencies *Dependencies) *cobra.Command {
	var q string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search people by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ps := store.NewPersonStore(dependencies.Pool)
			people, err := ps.Search(cmd.Context(), q)
			if err != nil {
				return err
			}
			for _, p := range people {
				fmt.Printf("%d\t%s\t%s\thidden=%v\n", p.ID, p.DisplayName, p.DisambiguationTag, p.IsHidden)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&q, "q", "", "Query String (required)")
	_ = cmd.MarkFlagRequired("q")

	return cmd
}

func cmdPerson(dependencies *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "person",
		Short: "Person maintenance",
	}

	var personID int64
	var hidden bool

	cmdHide := &cobra.Command{
		Use:   "hide",
		Short: "Hide or unhide a person",
		RunE: func(cmd *cobra.Command, args []string) error {
			ps := store.NewPersonStore(dependencies.Pool)
			return ps.SetHidden(cmd.Context(), personID, hidden)
		},
	}
	cmdHide.Flags().Int64Var(&personID, "id", 0, "Person id (required)")
	cmdHide.Flags().BoolVar(&hidden, "hidden", true, "Hidden flag")
	_ = cmdHide.MarkFlagRequired("id")

	cmdPurge := &cobra.Command{
		Use:   "purge",
		Short: "Purge a person (delete images + data).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ps := store.NewPersonStore(dependencies.Pool)
			return ps.Purge(cmd.Context(), personID)
		},
	}
	cmdPurge.Flags().Int64Var(&personID, "id", 0, "Person id (required)")
	_ = cmdPurge.MarkFlagRequired("id")

	cmd.AddCommand(cmdHide, cmdPurge)
	return cmd
}

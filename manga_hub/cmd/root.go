package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go_mangahub/manga_hub/cmd/api-server"
	"go_mangahub/manga_hub/cmd/mangahub"
)

// base "mangahub" command
var rootCmd = &cobra.Command{
	Use:   "mangahub",
	Short: "Mangahub is a tool to track your manga progress",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add server command to root command
	rootCmd.AddCommand(apiserver.ServerCmd) //

	// Add auth command to root command
	rootCmd.AddCommand(mangahub.AuthCmd) //

	// Add manga command to root command
	rootCmd.AddCommand(mangahub.MangaCmd)

	// Add library command to root command
	rootCmd.AddCommand(mangahub.LibraryCmd)

	// Add progress command to root command
	rootCmd.AddCommand(mangahub.ProgressCmd)

}

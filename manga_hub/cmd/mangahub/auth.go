package mangahub

import (
	"fmt"

	"github.com/spf13/cobra"
)



var username string

// subcommand "auth"
var AuthCmd = &cobra.Command{
	Use: "auth",
	Short: "Register an account/Log into an account",
}

var loginCmd = &cobra.Command{
	Use: "login",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Login user: %s...\n", username)
	},
}

func init() {

	// add login to auth command
	AuthCmd.AddCommand(loginCmd)

	// define flags
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "your username")
	loginCmd.MarkFlagRequired("username")
}

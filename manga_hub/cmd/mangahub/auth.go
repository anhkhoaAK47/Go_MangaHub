package mangahub

import (
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"io"
	"net/http"
	"os"

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

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "Show current login status and user information",
	Run: func(cmd *cobra.Command, args []string) {
		// read the saved token
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in.")
			fmt.Println("Try: mangahub auth login --username <username>")
			return
		}

		token := string(tokenData)

		// create GET request to server
		client := &http.Client{}
		req, err := http.NewRequest("GET", "http://localhost:8080/auth/status", nil)

		if err != nil {
			fmt.Printf("❌ Error sending GET request to /auth/status: %v\n", err)
			return
		}

		// add jwt token to the authorization header
		req.Header.Add("Authorization", "Bearer " + token)

		// send the request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err.Error())
			return
		}
		defer resp.Body.Close()

		// read the response 
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("❌ Session expired or invalid")
			fmt.Println("Please login again")

			return
		}
		
		if resp.StatusCode != http.StatusOK {
			fmt.Println("❌ Error: ",err.Error())
			return
		}

		// convert JSON into string
		var result models.StatusResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("❌ Error parsing server response")
			return
		}

		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("User ID: %s\n", result.User.ID)
		fmt.Printf("Username: %s\n", result.User.Username)
		fmt.Printf("Created At: %s\n", result.User.CreatedAt)
	},
}

func init() {

	// add login to auth command
	AuthCmd.AddCommand(loginCmd)

	// add status to auth command
	AuthCmd.AddCommand(statusCmd)

	// define flags login
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "your username")
	loginCmd.MarkFlagRequired("username")

	
}

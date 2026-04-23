package mangahub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/david_mbuvi/go_asterisks" // pkg to hide password input
)



var username string

// subcommand "auth"
var AuthCmd = &cobra.Command{
	Use: "auth",
	Short: "Register an account/Log into an account",
}

// Logic goes here
// loginCmd
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log into your account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Logging in as %s...\n", username)

		fmt.Print("Enter password: ")
		password, err := go_asterisks.GetUsersPassword("", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Printf("❌ Error: %s\n", err.Error())
			return
		}

		payload := map[string]string{
			"username": username,
			"password": string(password),
		}
		jsonData, _ := json.Marshal(payload)

		resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("❌ Server connection error. Is the server running?")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil {
				if errMsg, ok := result["error"].(string); ok {
					fmt.Printf("❌ %s\n", errMsg)
					if suggestion, ok := result["suggestion"].(string); ok {
						fmt.Printf("💡 %s\n", suggestion)
					}
					return
				}
			}
			// fallback
			fmt.Printf("❌ Login failed: %s\n", string(body))
			return
		}

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		token := result["token"].(string)
		os.WriteFile(".token", []byte(token), 0644)

		fmt.Printf("✅ Welcome back, %s!\n", username)
	},
}

// registerCmd
var registerCmd = &cobra.Command{
	Use:   "register [username]",
	Short: "Create a new account without an email",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Registering user: %s...\n", username)

		fmt.Print("Enter password: ")
		password, err := go_asterisks.GetUsersPassword("", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Printf("❌ Error: %s\n", err.Error())
			return
		}

		payload := map[string]string{
			"username": username,
			"password": string(password),
		}
		jsonData, _ := json.Marshal(payload)

		resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Println("❌ Server connection error.")
			return
		}
		defer resp.Body.Close()

		// ✅ Read body so we can print the server's error message
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			// ✅ Parse and print the "error" field from HandleRegister's JSON response
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil {
				if errMsg, ok := result["error"].(string); ok {
					fmt.Printf("❌ %s\n", errMsg)
					if suggestion, ok := result["suggestion"].(string); ok {
						fmt.Printf("💡 %s\n", suggestion)
					}
					return
				}
			}
			// fallback if JSON parsing fails
			fmt.Printf("❌ Registration failed: %s\n", string(body))
			return
		}

		fmt.Printf("✅ Account %s created! You can now login.\n", username)
	},
}

// logoutCmd
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear session",
	Run: func(cmd *cobra.Command, args []string) {
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("⚠️ You are already logged out.")
			return
		}

		client := &http.Client{}
		req, _ := http.NewRequest("POST", "http://localhost:8080/auth/logout", nil)
		req.Header.Add("Authorization", "Bearer "+string(tokenData))

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}

		os.Remove(".token")
		fmt.Println("✅ Logged out successfully and session cleared.")
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
		req, err := http.NewRequest("GET", "http://localhost:8080/auth/check", nil)

		if err != nil {
			fmt.Printf("❌ Error sending GET request to /auth/check: %v\n", err)
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
			fmt.Printf("❌ Server returned error: %s (Status: %d)\n", string(body), resp.StatusCode)
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


var changePasswordCmd = &cobra.Command{
	Use: "change-password",
	Short: "Change your mangahub's account password",
	Run: func(cmd *cobra.Command, args []string) {
		// prompt for changing password
		fmt.Print("Enter current password: ")
    	currentPassword, err := go_asterisks.GetUsersPassword("", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Printf("❌ Error: %s", err.Error())
			return
		}
		
		fmt.Print("Enter new password: ")
		newPassword, err := go_asterisks.GetUsersPassword("", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Printf("❌ Error: %s", err.Error())
			return
		}
		
		// read the saved token
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in.")
			fmt.Println("Try: mangahub auth login --username <username>")
			return		
		}

		token := string(tokenData)

		// create payload from user input
		input, _ := json.Marshal(map[string]string{
			"current_password": string(currentPassword),
			"new_password": string(newPassword),
		})
		payload := bytes.NewBuffer(input)

		// send PUT request to server
		client := &http.Client{}
		req, err := http.NewRequest("PUT", "http://localhost:8080/auth/change-password", payload)
		if err != nil {
			fmt.Println("❌ Failed to create PUT request")
			return
		}

		// add jwt token to authorization header
		req.Header.Add("Authorization", "Bearer " + token)
		req.Header.Set("Content-Type", "application/json")

		// send the request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Failed to send PUT request to server")
			return
		}

		// handle errors
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("❌ Failed to change password: %s\n", string(body))
			return
		} else {
			fmt.Println("✅ Password changed successfully!")
			return
		}
	},
}

func init() {

	// add login to auth command
	AuthCmd.AddCommand(loginCmd)

	// add register to auth command
	AuthCmd.AddCommand(registerCmd)

	// add logout to auth command
	AuthCmd.AddCommand(logoutCmd)
	
	// add status to auth command
	AuthCmd.AddCommand(statusCmd)

	// add change-password to auth command
	AuthCmd.AddCommand(changePasswordCmd)

	// define flags login
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "your username")
	loginCmd.MarkFlagRequired("username")


	// define flags register
	registerCmd.Flags().StringVarP(&username, "username", "u", "", "your username")
	registerCmd.MarkFlagRequired("username")

	
}

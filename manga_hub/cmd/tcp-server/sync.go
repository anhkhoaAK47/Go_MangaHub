package tcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var SyncCmd = &cobra.Command{
	Use: "sync",
	Short: "Manage real-time progress synchronization",
}

var connectCmd = &cobra.Command{
	Use: "connect",
	Short: "Connect to TCP server",
	Run: func(cmd *cobra.Command, args []string) {
		// Read token file to check if logged in
		tokenData, err := os.ReadFile(".token")
		if err != nil {
			fmt.Println("❌ Not logged in. Please run: mangahub auth login --username <username>")
			return
		}
	
		_ = tokenData // token used for auth context

		// laod user info from .session file
		userID, username := loadUserSession()
		if userID == "" {
			fmt.Println("❌ Could not load user session. Please login again.")
			return
		}

		fmt.Println("Connecting to TCP sync server at localhost:9090...")

		// open TCP connection to the server
		conn, err := net.Dial("tcp", "localhost:9090")
		if err != nil {
			fmt.Println("❌ Failed to connect to TCP sync server.")
			fmt.Println("   Is the server running? Try: mangahub server start")
			return
		}
		defer conn.Close()

		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		// Send registration message
		reg := map[string]string{
			"user_id": userID,
			"username": username,
			"device": "cli",
		}

		regData, _ := json.Marshal(reg)
		rw.Write(regData)
		rw.WriteByte('\n')
		rw.Flush()

		// Read welcome message
		line, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("❌ Failed to read server response:", err)
			return
		}
		var ack map[string]interface{}
		if err := json.Unmarshal([]byte(line), &ack); err == nil {
			fmt.Printf("✅ %s\n", ack["message"])
		}

		fmt.Printf("\nConnection Details:\n")
		fmt.Printf("  Server:     tcp://localhost:9090\n")
		fmt.Printf("  User:       %s\n", username)
		fmt.Printf("  Connected:  %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf("\nReal-time sync is now active. Listening for updates...\n")
		fmt.Printf("Press Ctrl+C to disconnect.\n\n")

		// save connection state
		os.WriteFile(".tcp_connected", []byte("connected"), 0644)
		defer os.Remove(".tcp_connected")

		// keep connection alive
		select {}
	},
}

var disconnectCmd = &cobra.Command{
	Use: "disconnect",
	Short: "Disconnect from TCP server",
	Run: func(cmd *cobra.Command, args []string) {
		// check if tcp_connected file exists
		if _, err := os.Stat(".tcp_connected"); os.IsNotExist(err) {
			fmt.Println("⚠️  Not currently connected to sync server.")
			return
		}

		// remove if exists
		os.Remove(".tcp_connected")
		fmt.Println("✅ Disconnected from sync server.")
	},
}


var statusCmd = &cobra.Command{
	Use: "status",
	Short: "Check TCP server connection status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TCP Server Status:")

		// Reach the TCP server
		conn, err := net.DialTimeout("tcp", "localhost:9090", 2*time.Second)
		if err != nil {
			fmt.Println("  Connection: ✗ Not reachable")
			fmt.Println("  Hint: Start the server with: mangahub server start")
			return
		}
		conn.Close()

		// check if this session is actively connected
		_, connected := os.Stat(".tcp_connected")
		if connected == nil {
			fmt.Println("  Connection: ✅ Active")
		} else {
			fmt.Println("  Connection: ⚠️  Server reachable but not connected")
			fmt.Println("  Run: mangahub sync connect")
		}

		fmt.Printf("  Server:     tcp://localhost:9090\n")
		fmt.Printf("  Checked at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	},
}


var monitorCmd = &cobra.Command{
	Use: "monitor",
	Short: "Watch real-time sync updates from all devices",
	Run: func(cmd *cobra.Command, args []string) {
		// verify user session
		userID, username := loadUserSession()
		if userID == "" {
			fmt.Println("❌ Not logged in. Please run: mangahub auth login --username <username>")
			return
		}

		// connect to TCP server
		conn, err := net.Dial("tcp", "localhost:9090")
		if err != nil {
			fmt.Println("❌ Failed to connect to TCP sync server.")
			return
		}
		defer conn.Close()

		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		// register with device type monitor
		reg := map[string]string{
			"user_id": userID,
			"username": username,
			"device": "monitor",
		}
		regData, _ := json.Marshal(reg)
		rw.Write(regData)
		rw.WriteByte('\n')
		rw.Flush()

		fmt.Println("Monitoring real-time sync updates... (Press Ctrl+C to exit)")

		// skip welcome ack
		rw.ReadString('\n')

		// print all broadcasting message
		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				break
			}
			var update map[string]interface{}
			if err := json.Unmarshal([]byte(line), &update); err != nil {
				continue
			}

			ts := time.Now().Format("15:04:05")
			fmt.Printf("[%s] ← %s updated: %s → Chapter %.0f\n",
				ts,
				update["username"],
				update["manga_id"],
				update["chapter"],
			)
		}
	},
}


func loadUserSession() (string, string) {
	data, err := os.ReadFile(".session")
	if err != nil {
		return "", ""
	}

	var session map[string]string
	if err := json.Unmarshal(data, &session); err != nil {
		return "", ""
	}

	return session["user_id"], session["username"]
}


func init() {
	// add connect command
	SyncCmd.AddCommand(connectCmd)

	// add disconnect command
	SyncCmd.AddCommand(disconnectCmd)

	// add status command
	SyncCmd.AddCommand(statusCmd)

	// add monitor command
	SyncCmd.AddCommand(monitorCmd)
}
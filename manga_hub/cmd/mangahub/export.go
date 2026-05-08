package mangahub

import (
	"archive/tar"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/utils"
	"os"

	"github.com/spf13/cobra"
)

var exportFormat string
var exportOutput string


var ExportCmd = &cobra.Command{
	Use: "export",
	Short: "Export the MangaHub database",
}

var libraryCmd = &cobra.Command{
	Use: "library",
	Short: "Export the user's manga library",
	Run: func(cmd *cobra.Command, args []string) {
		data := utils.FetchData("/users/export/library")
		SaveToFile(data, "library")
	},
}

var progressCmd = &cobra.Command{
	Use: "progress",
	Short: "Export the reading history",
	Run: func(cmd *cobra.Command, args []string) {
		data := utils.FetchData("/users/export/progress")
		SaveToFile(data, "progress")
	},
}

var allCmd =  &cobra.Command{
	Use: "all",
	Short: "Full data export",
	Run: func(cmd *cobra.Command, args []string) {
		libData := utils.FetchData("/users/export/library")
		progData := utils.FetchData("/users/export/progress")

		CreateTarball(libData, progData, exportOutput)
	},
}

func SaveToFile(data []byte, typeName string) {
	if exportFormat == "csv" {
		ConvertToCSV(data, typeName, exportOutput)
	} else {
		// default: JSON
		os.WriteFile(exportOutput, data, 0644)
		fmt.Printf("✅ Exported %s to %s (JSON)\n", typeName, exportFormat)
	}
}

func ConvertToCSV(data []byte, typeName string, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if typeName == "library" {
		writer.Write([]string{
			"MangaID", "Title", "Status", "Chapter",
		})
		entries := result["entries"].([]interface{})
		for _, e := range entries {
			m := e.(map[string]interface{})
			writer.Write([]string{
				m["manga_id"].(string),
				m["title"].(string),
				m["status"].(string),
				fmt.Sprintf("%.0f", m["current_chapter"]),
			})
		}
	} else {
		// progress history
		writer.Write([]string{
			"MangaID", "Title", "Status", "Chapter",
		})
		history := result["history"].([]interface{})
		for _, h := range history {
			m := h.(map[string]interface{})
			writer.Write([]string{
				m["manga_id"].(string), 
				fmt.Sprintf("%.0f", m["current_chapter"]), 
				m["notes"].(string), 
				m["updated_at"].(string),
		})
	}
	fmt.Printf("Exported %s to %s (CSV)\n", typeName, filename)
	}
}


func CreateTarball(lib []byte, prog []byte, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	files := map[string][]byte{
		"library.json": lib,
		"progress.json": prog,
	}

	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(body))}
		tw.WriteHeader(hdr)
		tw.Write(body)
	}
	fmt.Printf("Full backup created: %s\n", filename)
}

func init() {
	// export library command
	ExportCmd.AddCommand(libraryCmd)

	// export progress command
	ExportCmd.AddCommand(progressCmd)

	// export all command
	ExportCmd.AddCommand(allCmd)

	// Flags
	ExportCmd.PersistentFlags().StringVarP(&exportFormat, "format", "f", "json", "Output format (json/csv)")
	ExportCmd.PersistentFlags().StringVarP(&exportOutput, "output", "o", "export.dat", "Output filename")
}
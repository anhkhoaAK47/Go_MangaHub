package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var exportFormat string
var exportOutput string

func FetchData(endpoint string) []byte {
	token, _ := os.ReadFile(".token")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:8080"+endpoint, nil)
	req.Header.Add("Authorization", "Bearer " + string(token))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ Failed to fetch data from server")
		os.Exit(1)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body
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
		entries := result["entries"].([]interface{})
		for _, e := range entries {
			m := e.(map[string]interface{})
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
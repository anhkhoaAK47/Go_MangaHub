package models

import ()


type MangaDexResponse struct {
	Data []struct {
		ID		string `json:"id"`
		Attributes struct {
			Title		map[string]string `json:"title"`
			Description map[string]string `json:"description"`
			Status		string			  `json:"status"`
			LastChapter string			  `json:"lastChapter"`
		} `json:"attributes"`
		Relationships []struct {
			Type		string `json:"type"`
			Attributes  struct {
				Name	string `json:"name"`
				Filename string `json:"filename"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}
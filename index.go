package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func buildIndex(root string) (mis []MangaInfo, cis []ChapterInfo, pis []PageInfo, err error) {
	type mangaInfo struct {
		Name     string `json:"name"`
		Chapters []struct {
			Name  string   `json:"name"`
			Dir   string   `json:"dir"`
			Pages []string `json:"pages"`
		} `json:"chapters"`
	}

	infos, err := ioutil.ReadDir(root)
	if err != nil {
		return
	}

	for _, info := range infos {
		if !info.IsDir() {
			continue
		}

		dir := filepath.Join(root, info.Name())

		var mangaInfo mangaInfo
		if err := parseFile(filepath.Join(dir, ".mangainfo"), &mangaInfo); err != nil {
			log.Print(err)
			continue
		}

		mi := MangaInfo{
			Name:       mangaInfo.Name,
			ID:         len(mis),
			ChapterIDs: make([]int, len(mangaInfo.Chapters)),
		}

		for i, ch := range mangaInfo.Chapters {
			ci := ChapterInfo{
				ID:      len(cis),
				MangaID: mi.ID,
				Name:    ch.Name,
				PageIDs: make([]int, len(ch.Pages)),
			}

			for j, pg := range ch.Pages {
				pi := PageInfo{
					ID:        len(pis),
					ChapterID: ci.ID,
					Path:      filepath.Join(dir, ch.Dir, pg),
				}
				ci.PageIDs[j] = pi.ID
				pis = append(pis, pi)
			}

			mi.ChapterIDs[i] = ci.ID
			cis = append(cis, ci)
		}

		mis = append(mis, mi)
	}

	return
}

type MangaInfo struct {
	ID         int
	Name       string
	ChapterIDs []int
}

type ChapterInfo struct {
	ID      int
	MangaID int
	Name    string
	PageIDs []int
}

type PageInfo struct {
	ID        int
	ChapterID int
	Path      string // absolute
}

func parseFile(filename string, v interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	return json.NewDecoder(f).Decode(v)
}

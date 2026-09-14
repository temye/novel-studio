package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var storeDataPath string
var storeSaveMu sync.Mutex

type storeSnapshot struct {
	Projects   []Project        `json:"projects"`
	Chapters   []Chapter        `json:"chapters"`
	World      []WorldSetting   `json:"world"`
	Characters []Character      `json:"characters"`
	Locations  []Location       `json:"locations"`
	Items      []Item           `json:"items"`
	Tasks      []AITask         `json:"tasks"`
	Versions   []ChapterVersion `json:"versions"`
	Next       []int            `json:"next"`
}

func (s *Store) load() error {
	data, err := os.ReadFile(storeDataPath)
	if err != nil {
		return err
	}
	var snapshot storeSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	if len(snapshot.Next) != 8 {
		return fmt.Errorf("invalid local data format")
	}
	s.projects, s.chapters, s.world, s.characters = snapshot.Projects, snapshot.Chapters, snapshot.World, snapshot.Characters
	s.locations, s.items, s.tasks, s.versions = snapshot.Locations, snapshot.Items, snapshot.Tasks, snapshot.Versions
	s.nextProject, s.nextChapter, s.nextWorld, s.nextCharacter = snapshot.Next[0], snapshot.Next[1], snapshot.Next[2], snapshot.Next[3]
	s.nextLocation, s.nextItem, s.nextTask, s.nextVersion = snapshot.Next[4], snapshot.Next[5], snapshot.Next[6], snapshot.Next[7]
	return nil
}

func (s *Store) save() error {
	storeSaveMu.Lock()
	defer storeSaveMu.Unlock()
	s.RLock()
	snapshot := storeSnapshot{Projects: s.projects, Chapters: s.chapters, World: s.world, Characters: s.characters, Locations: s.locations, Items: s.items, Tasks: s.tasks, Versions: s.versions, Next: []int{s.nextProject, s.nextChapter, s.nextWorld, s.nextCharacter, s.nextLocation, s.nextItem, s.nextTask, s.nextVersion}}
	s.RUnlock()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(storeDataPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(storeDataPath, data, 0o600)
}

func persistAfterMutation(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.Method != http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/") {
			_ = store.save()
		}
	})
}

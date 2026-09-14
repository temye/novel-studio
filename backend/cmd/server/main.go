package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Project struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Genre        string    `json:"genre"`
	TargetWords  int       `json:"target_words"`
	CurrentWords int       `json:"current_words"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Chapter struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Outline   string    `json:"outline"`
	Content   string    `json:"content"`
	WordCount int       `json:"word_count"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorldSetting struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	Category    string    `json:"category"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Character struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Personality string    `json:"personality"`
	Goal        string    `json:"goal"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Location struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Item struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type AITask struct {
	ID          int        `json:"id"`
	ProjectID   int        `json:"project_id"`
	ChapterID   int        `json:"chapter_id,omitempty"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Result      string     `json:"result,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
type ChapterVersion struct {
	ID        int       `json:"id"`
	ChapterID int       `json:"chapter_id"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	WordCount int       `json:"word_count"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	sync.RWMutex
	projects                                                                                          []Project
	chapters                                                                                          []Chapter
	world                                                                                             []WorldSetting
	characters                                                                                        []Character
	locations                                                                                         []Location
	items                                                                                             []Item
	tasks                                                                                             []AITask
	versions                                                                                          []ChapterVersion
	nextProject, nextChapter, nextWorld, nextCharacter, nextLocation, nextItem, nextTask, nextVersion int
}

var authMu sync.RWMutex
var users = map[string]string{"author@example.com": "password"}
var authSecret []byte

const tokenTTL = 24 * time.Hour

type aiRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float32             `json:"temperature"`
}
type aiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	loadDotEnv()
	secret := strings.TrimSpace(os.Getenv("AUTH_SECRET"))
	if secret == "" || strings.HasPrefix(secret, "replace-with-") {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("AUTH_SECRET must be configured in production")
		}
		secret = "novel-studio-development-secret"
		log.Print("warning: using development auth secret; set AUTH_SECRET before deployment")
	}
	authSecret = []byte(secret)
	store := &Store{nextProject: 1, nextChapter: 1, nextWorld: 1, nextCharacter: 1, nextLocation: 1, nextItem: 1, nextTask: 1, nextVersion: 1}
	storeDataPath = envOr("NOVEL_DATA_FILE", filepath.Join("data", "novel-studio.json"))
	if err := store.load(); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("could not load local data: %v; starting with demo data", err)
		}
		store.seed()
		if err := store.save(); err != nil {
			log.Printf("could not save initial data: %v", err)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", health)
	mux.HandleFunc("/api/ai/status", aiStatus)
	mux.HandleFunc("/api/auth/login", login)
	mux.HandleFunc("/api/auth/register", register)
	mux.HandleFunc("/api/projects", store.projectsHandler)
	mux.HandleFunc("/api/projects/", store.projectHandler)
	mux.HandleFunc("/api/chapters/", store.chapterHandler)
	mux.HandleFunc("/api/tasks/", store.taskHandler)
	mux.Handle("/", frontendHandler())
	port := os.Getenv("PORT")
	if port == "" {
		port = "9008"
	}
	server := &http.Server{Addr: ":" + port, Handler: cors(securityHeaders(limitedBody(logging(authMiddleware(persistAfterMutation(store, mux))))))}
	log.Printf("Novel Studio listening on http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}

func (s *Store) seed() {
	now := time.Now()
	s.projects = append(s.projects, Project{ID: s.nextProject, Title: "未命名的星港", Summary: "一座漂浮在深空中的城市，正在等待它的第一段故事。", Genre: "科幻", TargetWords: 100000, CurrentWords: 12480, Status: "writing", CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now})
	p := s.nextProject
	s.nextProject++
	for i, title := range []string{"抵达星港", "没有名字的信", "潮汐引擎"} {
		s.chapters = append(s.chapters, Chapter{ID: s.nextChapter, ProjectID: p, Number: i + 1, Title: title, Outline: "这一章需要建立场景，并让主角面对一个无法回避的选择。", WordCount: []int{3577, 2913, 0}[i], Status: []string{"done", "done", "outline"}[i], UpdatedAt: now})
		s.nextChapter++
	}
	s.world = append(s.world, WorldSetting{ID: s.nextWorld, ProjectID: p, Category: "核心设定", Name: "星港公约", Description: "所有抵达星港的船只必须交出一段记忆作为通行凭证。", UpdatedAt: now})
	s.nextWorld++
	s.characters = append(s.characters, Character{ID: s.nextCharacter, ProjectID: p, Name: "林澈", Role: "主角", Personality: "谨慎、敏锐，不轻易相信承诺。", Goal: "找回被星港扣留的记忆。", UpdatedAt: now})
	s.nextCharacter++
	s.locations = append(s.locations, Location{ID: s.nextLocation, ProjectID: p, Name: "第七码头", Description: "星港最外侧的旧式泊位，常年笼罩在蓝色雾气中。", UpdatedAt: now})
	s.nextLocation++
	s.items = append(s.items, Item{ID: s.nextItem, ProjectID: p, Name: "潮汐引擎", Description: "能够改变局部时间流速的未知装置。", UpdatedAt: now})
	s.nextItem++
}

func health(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
func aiStatus(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{"configured": os.Getenv("AI_API_KEY") != "", "provider": envOr("AI_PROVIDER", "demo"), "model": envOr("AI_MODEL", "gpt-4o-mini"), "base_url": envOr("AI_BASE_URL", "")})
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Account, Password string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		badRequest(w, "请求格式不正确")
		return
	}
	authMu.RLock()
	pass, ok := users[in.Account]
	authMu.RUnlock()
	if !ok || pass != in.Password {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "账号或密码错误"})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"token": issueToken(in.Account), "user": map[string]string{"name": in.Account}})
}
func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Account, Password string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Account) == "" || len(in.Password) < 6 {
		badRequest(w, "账号不能为空，密码至少 6 位")
		return
	}
	authMu.Lock()
	if _, exists := users[in.Account]; exists {
		authMu.Unlock()
		jsonResponse(w, http.StatusConflict, map[string]string{"error": "账号已存在"})
		return
	}
	users[in.Account] = in.Password
	authMu.Unlock()
	jsonResponse(w, http.StatusCreated, map[string]any{"token": issueToken(in.Account), "user": map[string]string{"name": in.Account}})
}

func issueToken(account string) string {
	expires := time.Now().Add(tokenTTL).Unix()
	raw := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%d", account, expires)))
	h := hmac.New(sha256.New, authSecret)
	h.Write([]byte(raw))
	return raw + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/api/auth/") || r.URL.Path == "/api/health" || r.URL.Path == "/api/ai/status" {
			next.ServeHTTP(w, r)
			return
		}
		parts := strings.Split(r.Header.Get("Authorization"), " ")
		if len(parts) != 2 || parts[0] != "Bearer" || !validToken(parts[1]) {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func validToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || len(authSecret) == 0 {
		return false
	}
	h := hmac.New(sha256.New, authSecret)
	h.Write([]byte(parts[0]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, h.Sum(nil)) {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	fields := strings.Split(string(payload), "|")
	if len(fields) != 2 || fields[0] == "" {
		return false
	}
	expires, err := strconv.ParseInt(fields[1], 10, 64)
	return err == nil && time.Now().Unix() < expires
}

func (s *Store) projectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/projects" {
		notFound(w)
		return
	}
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		jsonResponse(w, http.StatusOK, s.projects)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		Title, Summary, Genre string
		TargetWords           int `json:"target_words"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		badRequest(w, "请求格式不正确")
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		badRequest(w, "小说名称不能为空")
		return
	}
	if input.TargetWords == 0 {
		input.TargetWords = 100000
	}
	now := time.Now()
	s.Lock()
	p := Project{ID: s.nextProject, Title: input.Title, Summary: input.Summary, Genre: input.Genre, TargetWords: input.TargetWords, Status: "planning", CreatedAt: now, UpdatedAt: now}
	s.nextProject++
	s.projects = append(s.projects, p)
	s.Unlock()
	jsonResponse(w, http.StatusCreated, p)
}

func (s *Store) projectHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		notFound(w)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		notFound(w)
		return
	}
	s.RLock()
	exists := false
	for _, p := range s.projects {
		if p.ID == id {
			exists = true
			break
		}
	}
	s.RUnlock()
	if !exists {
		notFound(w)
		return
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodPut, http.MethodPatch:
			s.updateProject(w, r, id)
			return
		case http.MethodDelete:
			s.deleteProject(w, id)
			return
		}
	}
	if len(parts) == 5 && parts[3] == "generate" && r.Method == http.MethodPost {
		s.createTask(w, id, 0, parts[4])
		return
	}
	if len(parts) == 5 {
		if resourceID, parseErr := strconv.Atoi(parts[4]); parseErr == nil {
			s.resourceItemHandler(w, r, id, parts[3], resourceID)
			return
		}
	}
	if len(parts) == 4 && parts[3] == "analysis" && r.Method == http.MethodGet {
		s.analysis(w, id)
		return
	}
	if len(parts) == 4 && parts[3] == "export" && r.Method == http.MethodGet {
		s.exportProject(w, id, r.URL.Query().Get("format"))
		return
	}
	if len(parts) == 4 {
		switch parts[3] {
		case "tasks":
			s.projectTasks(w, id)
		case "chapters":
			s.chapterCollection(w, r, id)
		case "world":
			s.worldCollection(w, r, id)
		case "characters":
			s.characterCollection(w, r, id)
		case "locations":
			s.locationCollection(w, r, id)
		case "items":
			s.itemCollection(w, r, id)
		default:
			notFound(w)
		}
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.RLock()
	defer s.RUnlock()
	for _, p := range s.projects {
		if p.ID == id {
			jsonResponse(w, http.StatusOK, p)
			return
		}
	}
	notFound(w)
}

func (s *Store) resourceItemHandler(w http.ResponseWriter, r *http.Request, projectID int, kind string, resourceID int) {
	if kind != "world" && kind != "characters" && kind != "locations" && kind != "items" {
		notFound(w)
		return
	}
	if r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if r.Method == http.MethodDelete {
		s.Lock()
		removed := false
		switch kind {
		case "world":
			filtered := s.world[:0]
			for _, x := range s.world {
				if x.ProjectID == projectID && x.ID == resourceID {
					removed = true
					continue
				}
				filtered = append(filtered, x)
			}
			s.world = filtered
		case "characters":
			filtered := s.characters[:0]
			for _, x := range s.characters {
				if x.ProjectID == projectID && x.ID == resourceID {
					removed = true
					continue
				}
				filtered = append(filtered, x)
			}
			s.characters = filtered
		case "locations":
			filtered := s.locations[:0]
			for _, x := range s.locations {
				if x.ProjectID == projectID && x.ID == resourceID {
					removed = true
					continue
				}
				filtered = append(filtered, x)
			}
			s.locations = filtered
		case "items":
			filtered := s.items[:0]
			for _, x := range s.items {
				if x.ProjectID == projectID && x.ID == resourceID {
					removed = true
					continue
				}
				filtered = append(filtered, x)
			}
			s.items = filtered
		}
		s.Unlock()
		if !removed {
			notFound(w)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"deleted": true, "id": resourceID})
		return
	}
	var in struct{ Category, Name, Description, Role, Personality, Goal string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		badRequest(w, "请求格式不正确")
		return
	}
	s.Lock()
	defer s.Unlock()
	now := time.Now()
	switch kind {
	case "world":
		for i := range s.world {
			if s.world[i].ProjectID == projectID && s.world[i].ID == resourceID {
				if in.Category != "" {
					s.world[i].Category = in.Category
				}
				if in.Name != "" {
					s.world[i].Name = strings.TrimSpace(in.Name)
				}
				if in.Description != "" {
					s.world[i].Description = in.Description
				}
				s.world[i].UpdatedAt = now
				jsonResponse(w, http.StatusOK, s.world[i])
				return
			}
		}
	case "characters":
		for i := range s.characters {
			if s.characters[i].ProjectID == projectID && s.characters[i].ID == resourceID {
				if in.Name != "" {
					s.characters[i].Name = strings.TrimSpace(in.Name)
				}
				if in.Role != "" {
					s.characters[i].Role = in.Role
				}
				if in.Personality != "" {
					s.characters[i].Personality = in.Personality
				}
				if in.Goal != "" {
					s.characters[i].Goal = in.Goal
				}
				s.characters[i].UpdatedAt = now
				jsonResponse(w, http.StatusOK, s.characters[i])
				return
			}
		}
	case "locations":
		for i := range s.locations {
			if s.locations[i].ProjectID == projectID && s.locations[i].ID == resourceID {
				if in.Name != "" {
					s.locations[i].Name = strings.TrimSpace(in.Name)
				}
				if in.Description != "" {
					s.locations[i].Description = in.Description
				}
				s.locations[i].UpdatedAt = now
				jsonResponse(w, http.StatusOK, s.locations[i])
				return
			}
		}
	case "items":
		for i := range s.items {
			if s.items[i].ProjectID == projectID && s.items[i].ID == resourceID {
				if in.Name != "" {
					s.items[i].Name = strings.TrimSpace(in.Name)
				}
				if in.Description != "" {
					s.items[i].Description = in.Description
				}
				s.items[i].UpdatedAt = now
				jsonResponse(w, http.StatusOK, s.items[i])
				return
			}
		}
	}
	notFound(w)
}

func (s *Store) updateProject(w http.ResponseWriter, r *http.Request, projectID int) {
	var in struct {
		Title       string `json:"title"`
		Summary     string `json:"summary"`
		Genre       string `json:"genre"`
		TargetWords int    `json:"target_words"`
		Status      string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		badRequest(w, "请求格式不正确")
		return
	}
	s.Lock()
	defer s.Unlock()
	for i := range s.projects {
		if s.projects[i].ID != projectID {
			continue
		}
		if strings.TrimSpace(in.Title) != "" {
			s.projects[i].Title = strings.TrimSpace(in.Title)
		}
		if in.Summary != "" {
			s.projects[i].Summary = in.Summary
		}
		if in.Genre != "" {
			s.projects[i].Genre = in.Genre
		}
		if in.TargetWords > 0 {
			s.projects[i].TargetWords = in.TargetWords
		}
		if in.Status != "" {
			s.projects[i].Status = in.Status
		}
		s.projects[i].UpdatedAt = time.Now()
		jsonResponse(w, http.StatusOK, s.projects[i])
		return
	}
	notFound(w)
}

func (s *Store) deleteProject(w http.ResponseWriter, projectID int) {
	s.Lock()
	defer s.Unlock()
	projects := s.projects[:0]
	for _, p := range s.projects {
		if p.ID != projectID {
			projects = append(projects, p)
		}
	}
	s.projects = projects
	chapters := s.chapters[:0]
	chapterIDs := map[int]bool{}
	for _, c := range s.chapters {
		if c.ProjectID == projectID {
			chapterIDs[c.ID] = true
			continue
		}
		chapters = append(chapters, c)
	}
	s.chapters = chapters
	world := s.world[:0]
	for _, x := range s.world {
		if x.ProjectID != projectID {
			world = append(world, x)
		}
	}
	s.world = world
	characters := s.characters[:0]
	for _, x := range s.characters {
		if x.ProjectID != projectID {
			characters = append(characters, x)
		}
	}
	s.characters = characters
	locations := s.locations[:0]
	for _, x := range s.locations {
		if x.ProjectID != projectID {
			locations = append(locations, x)
		}
	}
	s.locations = locations
	items := s.items[:0]
	for _, x := range s.items {
		if x.ProjectID != projectID {
			items = append(items, x)
		}
	}
	s.items = items
	tasks := s.tasks[:0]
	for _, x := range s.tasks {
		if x.ProjectID != projectID {
			tasks = append(tasks, x)
		}
	}
	s.tasks = tasks
	versions := s.versions[:0]
	for _, x := range s.versions {
		if !chapterIDs[x.ChapterID] {
			versions = append(versions, x)
		}
	}
	s.versions = versions
	jsonResponse(w, http.StatusOK, map[string]any{"deleted": true, "project_id": projectID})
}

func (s *Store) projectTasks(w http.ResponseWriter, projectID int) {
	s.RLock()
	defer s.RUnlock()
	out := []AITask{}
	for _, t := range s.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	jsonResponse(w, http.StatusOK, out)
}

func (s *Store) chapterCollection(w http.ResponseWriter, r *http.Request, projectID int) {
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		out := []Chapter{}
		for _, c := range s.chapters {
			if c.ProjectID == projectID {
				out = append(out, c)
			}
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Title, Outline string
		WordCount      int `json:"word_count"`
		Status         string
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Title) == "" {
		badRequest(w, "章节标题不能为空")
		return
	}
	s.Lock()
	defer s.Unlock()
	number := 1
	for _, c := range s.chapters {
		if c.ProjectID == projectID && c.Number >= number {
			number = c.Number + 1
		}
	}
	now := time.Now()
	c := Chapter{ID: s.nextChapter, ProjectID: projectID, Number: number, Title: in.Title, Outline: in.Outline, WordCount: in.WordCount, Status: in.Status, UpdatedAt: now}
	if c.Status == "" {
		c.Status = "outline"
	}
	s.nextChapter++
	s.chapters = append(s.chapters, c)
	jsonResponse(w, http.StatusCreated, c)
}

func (s *Store) worldCollection(w http.ResponseWriter, r *http.Request, projectID int) {
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		out := []WorldSetting{}
		for _, x := range s.world {
			if x.ProjectID == projectID {
				out = append(out, x)
			}
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Category, Name, Description string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		badRequest(w, "设定名称不能为空")
		return
	}
	s.Lock()
	defer s.Unlock()
	x := WorldSetting{ID: s.nextWorld, ProjectID: projectID, Category: in.Category, Name: in.Name, Description: in.Description, UpdatedAt: time.Now()}
	s.nextWorld++
	s.world = append(s.world, x)
	jsonResponse(w, http.StatusCreated, x)
}
func (s *Store) characterCollection(w http.ResponseWriter, r *http.Request, projectID int) {
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		out := []Character{}
		for _, x := range s.characters {
			if x.ProjectID == projectID {
				out = append(out, x)
			}
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Name, Role, Personality, Goal string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		badRequest(w, "角色名称不能为空")
		return
	}
	s.Lock()
	defer s.Unlock()
	x := Character{ID: s.nextCharacter, ProjectID: projectID, Name: in.Name, Role: in.Role, Personality: in.Personality, Goal: in.Goal, UpdatedAt: time.Now()}
	s.nextCharacter++
	s.characters = append(s.characters, x)
	jsonResponse(w, http.StatusCreated, x)
}
func (s *Store) locationCollection(w http.ResponseWriter, r *http.Request, projectID int) {
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		out := []Location{}
		for _, x := range s.locations {
			if x.ProjectID == projectID {
				out = append(out, x)
			}
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Name, Description string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		badRequest(w, "地点名称不能为空")
		return
	}
	s.Lock()
	defer s.Unlock()
	x := Location{ID: s.nextLocation, ProjectID: projectID, Name: in.Name, Description: in.Description, UpdatedAt: time.Now()}
	s.nextLocation++
	s.locations = append(s.locations, x)
	jsonResponse(w, http.StatusCreated, x)
}
func (s *Store) itemCollection(w http.ResponseWriter, r *http.Request, projectID int) {
	if r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		out := []Item{}
		for _, x := range s.items {
			if x.ProjectID == projectID {
				out = append(out, x)
			}
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Name, Description string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		badRequest(w, "道具名称不能为空")
		return
	}
	s.Lock()
	defer s.Unlock()
	x := Item{ID: s.nextItem, ProjectID: projectID, Name: in.Name, Description: in.Description, UpdatedAt: time.Now()}
	s.nextItem++
	s.items = append(s.items, x)
	jsonResponse(w, http.StatusCreated, x)
}
func (s *Store) chapterHandler(w http.ResponseWriter, r *http.Request) {
	raw := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/chapters/"), "/")
	parts := strings.Split(raw, "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		notFound(w)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		s.deleteChapter(w, id)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.RLock()
		defer s.RUnlock()
		for _, c := range s.chapters {
			if c.ID == id {
				jsonResponse(w, http.StatusOK, c)
				return
			}
		}
		notFound(w)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		if parts[1] == "check" {
			s.consistencyCheck(w, id)
			return
		}
		if parts[1] == "move" {
			s.moveChapter(w, r, id)
			return
		}
		s.createTask(w, 0, id, parts[1])
		return
	}
	if len(parts) == 2 && parts[1] == "versions" && r.Method == http.MethodGet {
		s.versionsForChapter(w, id)
		return
	}
	if len(parts) == 3 && parts[1] == "versions" && parts[2] != "" && r.Method == http.MethodPost {
		versionID, err := strconv.Atoi(parts[2])
		if err != nil {
			notFound(w)
			return
		}
		s.restoreVersion(w, id, versionID)
		return
	}
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Title     *string `json:"title"`
		Outline   *string `json:"outline"`
		Content   *string `json:"content"`
		Status    *string `json:"status"`
		WordCount *int    `json:"word_count"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		badRequest(w, "请求格式不正确")
		return
	}
	s.Lock()
	defer s.Unlock()
	for i := range s.chapters {
		if s.chapters[i].ID == id {
			if in.Title != nil && strings.TrimSpace(*in.Title) != "" {
				s.chapters[i].Title = strings.TrimSpace(*in.Title)
			}
			if in.Outline != nil {
				s.chapters[i].Outline = *in.Outline
			}
			if in.Content != nil {
				s.chapters[i].Content = *in.Content
				s.versions = append(s.versions, ChapterVersion{ID: s.nextVersion, ChapterID: id, Content: *in.Content, Source: "manual", WordCount: len([]rune(*in.Content)), CreatedAt: time.Now()})
				s.nextVersion++
			}
			if in.Status != nil && *in.Status != "" {
				s.chapters[i].Status = *in.Status
			}
			if in.WordCount != nil && *in.WordCount >= 0 {
				s.chapters[i].WordCount = *in.WordCount
			}
			s.chapters[i].UpdatedAt = time.Now()
			s.recalculateProjectWords(s.chapters[i].ProjectID)
			jsonResponse(w, http.StatusOK, s.chapters[i])
			return
		}
	}
	notFound(w)
}

func (s *Store) moveChapter(w http.ResponseWriter, r *http.Request, chapterID int) {
	var in struct {
		ToNumber int `json:"to_number"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.ToNumber < 1 {
		badRequest(w, "目标章节序号不正确")
		return
	}
	s.Lock()
	defer s.Unlock()
	current := -1
	projectID := 0
	for i := range s.chapters {
		if s.chapters[i].ID == chapterID {
			current = i
			projectID = s.chapters[i].ProjectID
			break
		}
	}
	if current < 0 {
		notFound(w)
		return
	}
	target := -1
	for i := range s.chapters {
		if s.chapters[i].ProjectID == projectID && s.chapters[i].Number == in.ToNumber {
			target = i
			break
		}
	}
	if target >= 0 && target != current {
		s.chapters[current].Number, s.chapters[target].Number = s.chapters[target].Number, s.chapters[current].Number
		s.chapters[current].UpdatedAt = time.Now()
		s.chapters[target].UpdatedAt = time.Now()
	}
	jsonResponse(w, http.StatusOK, s.chapters[current])
}

func (s *Store) deleteChapter(w http.ResponseWriter, chapterID int) {
	s.Lock()
	defer s.Unlock()
	projectID := 0
	chapters := s.chapters[:0]
	for _, c := range s.chapters {
		if c.ID == chapterID {
			projectID = c.ProjectID
			continue
		}
		chapters = append(chapters, c)
	}
	if projectID == 0 {
		notFound(w)
		return
	}
	s.chapters = chapters
	versions := s.versions[:0]
	for _, v := range s.versions {
		if v.ChapterID != chapterID {
			versions = append(versions, v)
		}
	}
	s.versions = versions
	s.recalculateProjectWords(projectID)
	jsonResponse(w, http.StatusOK, map[string]any{"deleted": true, "chapter_id": chapterID})
}

func (s *Store) versionsForChapter(w http.ResponseWriter, chapterID int) {
	s.RLock()
	defer s.RUnlock()
	out := []ChapterVersion{}
	for _, v := range s.versions {
		if v.ChapterID == chapterID {
			out = append(out, v)
		}
	}
	jsonResponse(w, http.StatusOK, out)
}
func (s *Store) restoreVersion(w http.ResponseWriter, chapterID, versionID int) {
	s.Lock()
	defer s.Unlock()
	var content string
	for _, v := range s.versions {
		if v.ID == versionID && v.ChapterID == chapterID {
			content = v.Content
		}
	}
	if content == "" {
		notFound(w)
		return
	}
	now := time.Now()
	for i := range s.chapters {
		if s.chapters[i].ID == chapterID {
			s.chapters[i].Content = content
			s.chapters[i].WordCount = len([]rune(content))
			s.chapters[i].Status = "writing"
			s.chapters[i].UpdatedAt = now
			s.versions = append(s.versions, ChapterVersion{ID: s.nextVersion, ChapterID: chapterID, Content: content, Source: "restore", WordCount: len([]rune(content)), CreatedAt: now})
			s.nextVersion++
			s.recalculateProjectWords(s.chapters[i].ProjectID)
			jsonResponse(w, http.StatusOK, s.chapters[i])
			return
		}
	}
	notFound(w)
}

func (s *Store) recalculateProjectWords(projectID int) {
	total := 0
	for _, c := range s.chapters {
		if c.ProjectID == projectID {
			total += c.WordCount
		}
	}
	for i := range s.projects {
		if s.projects[i].ID == projectID {
			s.projects[i].CurrentWords = total
			s.projects[i].UpdatedAt = time.Now()
		}
	}
}
func (s *Store) analysis(w http.ResponseWriter, projectID int) {
	s.RLock()
	defer s.RUnlock()
	words, done, chapters, characters, world := 0, 0, 0, 0, 0
	for _, c := range s.chapters {
		if c.ProjectID == projectID {
			chapters++
			words += c.WordCount
			if c.Status == "done" {
				done++
			}
		}
	}
	for _, c := range s.characters {
		if c.ProjectID == projectID {
			characters++
		}
	}
	for _, x := range s.world {
		if x.ProjectID == projectID {
			world++
		}
	}
	rate := 0
	if chapters > 0 {
		rate = done * 100 / chapters
	}
	jsonResponse(w, http.StatusOK, map[string]any{"project_id": projectID, "chapters": chapters, "completed_chapters": done, "completion_rate": rate, "word_count": words, "characters": characters, "world_settings": world})
}

func (s *Store) consistencyCheck(w http.ResponseWriter, chapterID int) {
	s.RLock()
	defer s.RUnlock()
	var content string
	var projectID int
	for _, c := range s.chapters {
		if c.ID == chapterID {
			content = c.Content
			projectID = c.ProjectID
		}
	}
	warnings := []string{}
	if strings.TrimSpace(content) == "" {
		warnings = append(warnings, "正文为空，暂时无法检查一致性")
	}
	known := map[string]bool{}
	for _, c := range s.characters {
		if c.ProjectID == projectID {
			known[c.Name] = true
		}
	}
	for _, name := range []string{"林澈", "顾遥", "沈舟"} {
		if strings.Contains(content, name) && !known[name] {
			warnings = append(warnings, "正文出现未登记角色："+name)
		}
	}
	jsonResponse(w, http.StatusOK, map[string]any{"chapter_id": chapterID, "score": 100 - len(warnings)*15, "warnings": warnings, "checked_at": time.Now()})
}
func (s *Store) exportProject(w http.ResponseWriter, projectID int, format string) {
	s.RLock()
	defer s.RUnlock()
	title := "小说项目"
	var chapters []Chapter
	for _, p := range s.projects {
		if p.ID == projectID {
			title = p.Title
		}
	}
	for _, c := range s.chapters {
		if c.ProjectID == projectID {
			chapters = append(chapters, c)
		}
	}
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	for _, c := range chapters {
		fmt.Fprintf(&b, "## 第%d章 %s\n\n", c.Number, c.Title)
		if strings.TrimSpace(c.Content) != "" {
			b.WriteString(c.Content)
		} else {
			b.WriteString(c.Outline)
		}
		b.WriteString("\n\n")
	}
	if format == "txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	}
	ext := "novel-export.md"
	if format == "txt" {
		ext = "novel-export.txt"
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+ext+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (s *Store) createTask(w http.ResponseWriter, projectID, chapterID int, kind string) {
	// Resolve the project for chapter-level actions, which only carry a chapter ID.
	if chapterID > 0 && projectID == 0 {
		s.RLock()
		for _, c := range s.chapters {
			if c.ID == chapterID {
				projectID = c.ProjectID
				break
			}
		}
		s.RUnlock()
	}
	now := time.Now()
	s.Lock()
	task := AITask{ID: s.nextTask, ProjectID: projectID, ChapterID: chapterID, Type: kind, Status: "running", Progress: 5, CreatedAt: now}
	s.nextTask++
	s.tasks = append(s.tasks, task)
	s.Unlock()
	jsonResponse(w, http.StatusAccepted, task)
	go s.finishTask(task.ID, kind, projectID, chapterID)
}
func (s *Store) finishTask(id int, kind string, projectID, chapterID int) {
	for _, p := range []int{25, 55, 85} {
		time.Sleep(300 * time.Millisecond)
		s.Lock()
		for i := range s.tasks {
			if s.tasks[i].ID == id {
				s.tasks[i].Progress = p
			}
		}
		s.Unlock()
	}
	now := time.Now()
	result := s.generateWithProvider(kind, projectID, chapterID)
	s.Lock()
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Progress = 100
			s.tasks[i].Status = "completed"
			s.tasks[i].Result = result
			s.tasks[i].CompletedAt = &now
		}
	}
	// A generated result remains an unsaved editor draft; it must not affect
	// chapter status or word statistics until the author saves it.

	if kind == "world" {
		s.world = append(s.world, WorldSetting{ID: s.nextWorld, ProjectID: projectID, Category: "AI 生成", Name: "新的世界设定", Description: result, UpdatedAt: now})
		s.nextWorld++
	}
	if kind == "characters" {
		s.characters = append(s.characters, Character{ID: s.nextCharacter, ProjectID: projectID, Name: "新角色", Role: "AI 生成", Personality: result, UpdatedAt: now})
		s.nextCharacter++
	}
	if kind == "outline" && projectID > 0 {
		number := 1
		for _, c := range s.chapters {
			if c.ProjectID == projectID && c.Number >= number {
				number = c.Number + 1
			}
		}
		s.chapters = append(s.chapters, Chapter{ID: s.nextChapter, ProjectID: projectID, Number: number, Title: "AI 生成章节", Outline: result, Status: "outline", UpdatedAt: now})
		s.nextChapter++
	}
	s.Unlock()
	if err := s.save(); err != nil {
		log.Printf("could not persist AI task result: %v", err)
	}
}

func (s *Store) generateWithProvider(kind string, projectID, chapterID int) string {
	result := "已生成一份可继续编辑的演示结果。"
	if kind == "outline" {
		result = "本章从一个异常信号开始，主角在调查过程中发现新的线索，并被迫作出选择。"
	}
	if kind == "generate" || kind == "continue" || kind == "rewrite" {
		result = "雾气沿着第七码头的舷窗缓慢退去。林澈抬起手腕，潮汐引擎在沉默中亮起，像一颗迟到的星。"
	}
	base := strings.TrimRight(os.Getenv("AI_BASE_URL"), "/")
	key := os.Getenv("AI_API_KEY")
	if base == "" || key == "" {
		return result
	}
	context := s.aiContext(projectID, chapterID)
	prompt := map[string]string{"world": "请为当前小说生成一条可执行的世界观设定。只输出设定正文。", "characters": "请为当前小说生成一个核心角色档案，包含姓名、身份、性格、目标和秘密。只输出档案。", "outline": "请为当前小说生成一章剧情大纲，包含开场、冲突、转折和结尾钩子。只输出大纲。", "generate": "请根据当前小说设定生成一段章节正文，保持人物和世界观一致。只输出正文。", "continue": "请续写当前章节正文，承接上下文并制造新的推进。只输出正文。", "rewrite": "请润色当前章节正文，提升画面感和节奏，不改变事实。只输出正文。"}[kind] + "\n\n当前项目上下文：\n" + context
	body, _ := json.Marshal(aiRequest{Model: envOr("AI_MODEL", "gpt-4o-mini"), Messages: []map[string]string{{"role": "system", "content": "你是一个严谨的中文小说创作助手。"}, {"role": "user", "content": prompt}}, Temperature: .8})
	req, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return result
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return result
	}
	var out aiResponse
	if json.Unmarshal(data, &out) == nil && len(out.Choices) > 0 && strings.TrimSpace(out.Choices[0].Message.Content) != "" {
		return strings.TrimSpace(out.Choices[0].Message.Content)
	}
	return result
}
func (s *Store) aiContext(projectID, chapterID int) string {
	s.RLock()
	defer s.RUnlock()
	var b strings.Builder
	for _, p := range s.projects {
		if p.ID == projectID {
			fmt.Fprintf(&b, "小说：%s\n简介：%s\n", p.Title, p.Summary)
		}
	}
	b.WriteString("世界设定：\n")
	for _, x := range s.world {
		if x.ProjectID == projectID {
			fmt.Fprintf(&b, "- %s：%s\n", x.Name, x.Description)
		}
	}
	b.WriteString("角色：\n")
	for _, x := range s.characters {
		if x.ProjectID == projectID {
			fmt.Fprintf(&b, "- %s（%s）：%s\n", x.Name, x.Role, x.Personality)
		}
	}
	for _, c := range s.chapters {
		if c.ID == chapterID {
			fmt.Fprintf(&b, "当前章节：第%d章 %s\n大纲：%s\n正文：%s\n", c.Number, c.Title, c.Outline, c.Content)
		}
	}
	return b.String()
}
func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv() {
	candidates := []string{".env", filepath.Join("..", ".env")}
	for _, name := range candidates {
		data, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			pair := strings.SplitN(line, "=", 2)
			if len(pair) != 2 {
				continue
			}
			key := strings.TrimSpace(pair[0])
			value := strings.Trim(strings.TrimSpace(pair[1]), "\"'")
			if key != "" && os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
	}
}

func (s *Store) taskHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		notFound(w)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		notFound(w)
		return
	}
	s.RLock()
	var task AITask
	found := false
	for _, t := range s.tasks {
		if t.ID == id {
			task = t
			found = true
			break
		}
	}
	s.RUnlock()
	if !found {
		notFound(w)
		return
	}
	if len(parts) == 4 && parts[3] == "stream" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for i := 0; i < 240; i++ {
			s.RLock()
			for _, t := range s.tasks {
				if t.ID == id {
					task = t
				}
			}
			s.RUnlock()
			b, _ := json.Marshal(task)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
			if task.Status == "completed" || task.Status == "failed" {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		return
	}
	jsonResponse(w, http.StatusOK, task)
}

func frontendHandler() http.Handler {
	root, _ := os.Getwd()
	path := filepath.Join(root, "frontend")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(root, "..", "frontend")
	}
	return http.FileServer(http.Dir(path))
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func limitedBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Method != http.MethodGet {
			r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		}
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func badRequest(w http.ResponseWriter, msg string) {
	jsonResponse(w, http.StatusBadRequest, map[string]string{"error": msg})
}
func notFound(w http.ResponseWriter) {
	jsonResponse(w, http.StatusNotFound, map[string]string{"error": "资源不存在"})
}
func methodNotAllowed(w http.ResponseWriter) {
	jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "不支持的请求方法"})
}

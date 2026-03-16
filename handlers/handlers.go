package handlers

import (
        "bytes"
        "fmt"
        "html/template"
        "log"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "sort"
        "strings"
        "time"

        "repo/db"
)

// ─── Git tree types ──────────────────────────────────────────────────────────

type CommitInfo struct {
        Hash    string
        Short   string
        Message string
        Date    time.Time
}

type TreeEntry struct {
        Name   string
        Path   string
        IsDir  bool
        Commit CommitInfo
}

type Breadcrumb struct {
        Name string
        URL  string
}

// ─── Git helpers ─────────────────────────────────────────────────────────────

func repoDir(owner, name string) string {
        return filepath.Join("repos", owner, name+".git")
}

func gitIsEmpty(dir string) bool {
        return exec.Command("git", "-C", dir, "rev-parse", "HEAD").Run() != nil
}

func gitDefaultBranch(dir string) string {
        out, err := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "HEAD").Output()
        if err != nil {
                return "master"
        }
        return strings.TrimSpace(string(out))
}

func gitLastCommit(dir string) CommitInfo {
        out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%H|%s|%aI").Output()
        if err != nil {
                return CommitInfo{}
        }
        return parseCommitLine(strings.TrimSpace(string(out)))
}

func gitFileLastCommit(dir, branch, path string) CommitInfo {
        out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%H|%s|%aI", branch, "--", path).Output()
        if err != nil {
                return CommitInfo{}
        }
        return parseCommitLine(strings.TrimSpace(string(out)))
}

func parseCommitLine(s string) CommitInfo {
        if s == "" {
                return CommitInfo{}
        }
        parts := strings.SplitN(s, "|", 3)
        if len(parts) < 3 {
                return CommitInfo{}
        }
        h := strings.TrimSpace(parts[0])
        short := h
        if len(short) > 7 {
                short = short[:7]
        }
        t, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[2]))
        return CommitInfo{Hash: h, Short: short, Message: strings.TrimSpace(parts[1]), Date: t}
}

func gitListTree(dir, branch, subPath string) []TreeEntry {
        treeRef := branch
        if subPath != "" {
                treeRef = branch + ":" + subPath
        }
        out, err := exec.Command("git", "-C", dir, "ls-tree", treeRef).Output()
        if err != nil {
                return nil
        }
        var entries []TreeEntry
        for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
                if line == "" {
                        continue
                }
                tab := strings.Index(line, "\t")
                if tab < 0 {
                        continue
                }
                name := line[tab+1:]
                meta := strings.Fields(line[:tab])
                if len(meta) < 2 {
                        continue
                }
                isDir := meta[1] == "tree"
                var fullPath string
                if subPath == "" {
                        fullPath = name
                } else {
                        fullPath = subPath + "/" + name
                }
                entries = append(entries, TreeEntry{
                        Name:   name,
                        Path:   fullPath,
                        IsDir:  isDir,
                        Commit: gitFileLastCommit(dir, branch, fullPath),
                })
        }
        sort.Slice(entries, func(i, j int) bool {
                if entries[i].IsDir != entries[j].IsDir {
                        return entries[i].IsDir
                }
                return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
        })
        return entries
}

func gitFileContent(dir, branch, path string) (content string, isBinary bool, err error) {
        out, err := exec.Command("git", "-C", dir, "show", branch+":"+path).Output()
        if err != nil {
                return "", false, err
        }
        if bytes.IndexByte(out, 0) >= 0 {
                return "", true, nil
        }
        return string(out), false, nil
}

func buildBreadcrumbs(ownerName, repoName, subPath string) []Breadcrumb {
        base := "/" + ownerName + "/" + repoName
        if subPath == "" {
                return []Breadcrumb{{Name: repoName, URL: ""}}
        }
        crumbs := []Breadcrumb{{Name: repoName, URL: base}}
        parts := strings.Split(subPath, "/")
        for i, part := range parts {
                if i == len(parts)-1 {
                        crumbs = append(crumbs, Breadcrumb{Name: part, URL: ""})
                } else {
                        crumbs = append(crumbs, Breadcrumb{Name: part, URL: base + "/tree/" + strings.Join(parts[:i+1], "/")})
                }
        }
        return crumbs
}

var pageTemplates map[string]*template.Template

var reValidName = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// Init pre-parses each page template together with the shared layout so that
// {{define "content"}} blocks from different pages don't conflict.
func Init(templatesDir string) error {
        funcMap := template.FuncMap{
                "timeAgo": timeAgo,
		"add1":    func(i int) int { return i + 1 },
        }

        pages := []string{
                "home", "login", "register", "dashboard", "new_repo", "profile", "repo", "blob",
        }

        pageTemplates = make(map[string]*template.Template, len(pages))
        layoutPath := filepath.Join(templatesDir, "layout.html")

        for _, name := range pages {
                pagePath := filepath.Join(templatesDir, name+".html")
                t, err := template.New(name+".html").Funcs(funcMap).ParseFiles(layoutPath, pagePath)
                if err != nil {
                        return fmt.Errorf("parse %s: %w", name, err)
                }
                pageTemplates[name+".html"] = t
        }
        return nil
}

func timeAgo(t time.Time) string {
        d := time.Since(t)
        switch {
        case d < time.Minute:
                return "just now"
        case d < time.Hour:
                return fmt.Sprintf("%dm ago", int(d.Minutes()))
        case d < 24*time.Hour:
                return fmt.Sprintf("%dh ago", int(d.Hours()))
        case d < 30*24*time.Hour:
                return fmt.Sprintf("%dd ago", int(d.Hours()/24))
        default:
                return t.Format("Jan 2, 2006")
        }
}

// ─── session helpers ───────────────────────────────────────────────────────

const sessionCookie = "session"

func CurrentUser(r *http.Request) *db.User {
        c, err := r.Cookie(sessionCookie)
        if err != nil {
                return nil
        }
        u, _ := db.GetSessionUser(c.Value)
        return u
}

func setSession(w http.ResponseWriter, token string) {
        http.SetCookie(w, &http.Cookie{
                Name:     sessionCookie,
                Value:    token,
                Path:     "/",
                HttpOnly: true,
                MaxAge:   60 * 60 * 24 * 30,
                SameSite: http.SameSiteLaxMode,
        })
}

func clearSession(w http.ResponseWriter, r *http.Request) {
        c, err := r.Cookie(sessionCookie)
        if err == nil {
                _ = db.DeleteSession(c.Value)
        }
        http.SetCookie(w, &http.Cookie{
                Name:   sessionCookie,
                Value:  "",
                Path:   "/",
                MaxAge: -1,
        })
}

// ─── render helper ─────────────────────────────────────────────────────────

type PageData struct {
        User  *db.User
        Data  interface{}
        Flash string
        Error string
}

func render(w http.ResponseWriter, name string, data PageData) {
        t, ok := pageTemplates[name]
        if !ok {
                http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
                return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        if err := t.ExecuteTemplate(w, "layout", data); err != nil {
                log.Printf("template %s: %v", name, err)
                http.Error(w, "Internal server error", 500)
        }
}

// ─── Register ──────────────────────────────────────────────────────────────

func RegisterGET(w http.ResponseWriter, r *http.Request) {
        if CurrentUser(r) != nil {
                http.Redirect(w, r, "/dashboard", http.StatusFound)
                return
        }
        render(w, "register.html", PageData{})
}

func RegisterPOST(w http.ResponseWriter, r *http.Request) {
        username := strings.TrimSpace(r.FormValue("username"))
        email := strings.TrimSpace(r.FormValue("email"))
        password := r.FormValue("password")
        confirm := r.FormValue("confirm")

        errMsg := func(msg string) {
                render(w, "register.html", PageData{Error: msg, Data: map[string]string{
                        "Username": username,
                        "Email":    email,
                }})
        }

        if username == "" || email == "" || password == "" {
                errMsg("All fields are required.")
                return
        }
        if !reValidName.MatchString(username) {
                errMsg("Username may only contain letters, numbers, hyphens, underscores, and dots.")
                return
        }
        if len(username) < 3 || len(username) > 39 {
                errMsg("Username must be between 3 and 39 characters.")
                return
        }
        if password != confirm {
                errMsg("Passwords do not match.")
                return
        }
        if len(password) < 8 {
                errMsg("Password must be at least 8 characters.")
                return
        }

        u, err := db.CreateUser(username, email, password)
        if err != nil {
                if strings.Contains(err.Error(), "UNIQUE") {
                        errMsg("Username or email already in use.")
                } else {
                        errMsg("Could not create account. Please try again.")
                        log.Printf("create user: %v", err)
                }
                return
        }

        token, err := db.CreateSession(u.ID)
        if err != nil {
                errMsg("Account created but login failed. Please log in.")
                return
        }

        setSession(w, token)
        http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// ─── Login ─────────────────────────────────────────────────────────────────

func LoginGET(w http.ResponseWriter, r *http.Request) {
        if CurrentUser(r) != nil {
                http.Redirect(w, r, "/dashboard", http.StatusFound)
                return
        }
        render(w, "login.html", PageData{})
}

func LoginPOST(w http.ResponseWriter, r *http.Request) {
        username := strings.TrimSpace(r.FormValue("username"))
        password := r.FormValue("password")

        u, err := db.AuthenticateUser(username, password)
        if err != nil || u == nil {
                log.Printf("LOGIN FAIL: username=%q err=%v", username, err)
                render(w, "login.html", PageData{
                        Error: "Invalid username or password.",
                        Data:  map[string]string{"Username": username},
                })
                return
        }

        token, err := db.CreateSession(u.ID)
        if err != nil {
                log.Printf("LOGIN SESSION CREATE FAIL: user=%q err=%v", username, err)
                render(w, "login.html", PageData{Error: "Login failed. Please try again."})
                return
        }

        log.Printf("LOGIN OK: username=%q token=%s…", username, token[:8])
        setSession(w, token)
        http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// ─── Logout ────────────────────────────────────────────────────────────────

func Logout(w http.ResponseWriter, r *http.Request) {
        clearSession(w, r)
        http.Redirect(w, r, "/", http.StatusFound)
}

// ─── Home ──────────────────────────────────────────────────────────────────

func Home(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
                http.NotFound(w, r)
                return
        }
        u := CurrentUser(r)
        if u != nil {
                http.Redirect(w, r, "/dashboard", http.StatusFound)
                return
        }

        repos, _ := db.ListPublicRepositories()
        render(w, "home.html", PageData{User: u, Data: repos})
}

// ─── Dashboard ─────────────────────────────────────────────────────────────

func Dashboard(w http.ResponseWriter, r *http.Request) {
        c, cookieErr := r.Cookie(sessionCookie)
        if cookieErr != nil {
                log.Printf("DASHBOARD: no session cookie (%v)", cookieErr)
        } else {
                log.Printf("DASHBOARD: got cookie token=%s…", c.Value[:8])
        }
        u := CurrentUser(r)
        if u == nil {
                log.Printf("DASHBOARD: session not found, redirecting to /login")
                http.Redirect(w, r, "/login", http.StatusFound)
                return
        }

        repos, err := db.ListUserRepositories(u.Username)
        if err != nil {
                log.Printf("list repos: %v", err)
        }

        render(w, "dashboard.html", PageData{User: u, Data: repos})
}

// ─── New Repo ──────────────────────────────────────────────────────────────

func NewRepoGET(w http.ResponseWriter, r *http.Request) {
        u := CurrentUser(r)
        if u == nil {
                http.Redirect(w, r, "/login", http.StatusFound)
                return
        }
        render(w, "new_repo.html", PageData{User: u})
}

func NewRepoPOST(w http.ResponseWriter, r *http.Request) {
        u := CurrentUser(r)
        if u == nil {
                http.Redirect(w, r, "/login", http.StatusFound)
                return
        }

        name := strings.TrimSpace(r.FormValue("name"))
        desc := strings.TrimSpace(r.FormValue("description"))
        visibility := r.FormValue("visibility")
        isPrivate := visibility == "private"

        errMsg := func(msg string) {
                render(w, "new_repo.html", PageData{
                        User:  u,
                        Error: msg,
                        Data: map[string]interface{}{
                                "Name":        name,
                                "Description": desc,
                                "Private":     isPrivate,
                        },
                })
        }

        if name == "" {
                errMsg("Repository name is required.")
                return
        }
        if !reValidName.MatchString(name) {
                errMsg("Name may only contain letters, numbers, hyphens, underscores, and dots.")
                return
        }
        if len(name) > 100 {
                errMsg("Name must be 100 characters or less.")
                return
        }

        existing, _ := db.GetRepository(u.Username, name)
        if existing != nil {
                errMsg("You already have a repository with that name.")
                return
        }

        repo, err := db.CreateRepository(u.ID, name, desc, isPrivate)
        if err != nil {
                errMsg("Could not create repository. Please try again.")
                log.Printf("create repo: %v", err)
                return
        }

        if err := InitGitRepo(u.Username, repo.Name); err != nil {
                log.Printf("init git repo: %v", err)
        }

        http.Redirect(w, r, "/"+u.Username+"/"+repo.Name, http.StatusFound)
}

// ─── User Profile ──────────────────────────────────────────────────────────

func UserProfile(w http.ResponseWriter, r *http.Request, username string) {
        viewer := CurrentUser(r)

        owner, err := db.GetUserByUsername(username)
        if err != nil || owner == nil {
                http.NotFound(w, r)
                return
        }

        repos, _ := db.ListUserRepositories(username)

        if viewer == nil || viewer.Username != username {
                var public []db.Repository
                for _, repo := range repos {
                        if !repo.IsPrivate {
                                public = append(public, repo)
                        }
                }
                repos = public
        }

        type profileData struct {
                Owner *db.User
                Repos []db.Repository
        }

        render(w, "profile.html", PageData{
                User: viewer,
                Data: profileData{Owner: owner, Repos: repos},
        })
}

// ─── Repo View ─────────────────────────────────────────────────────────────

type repoViewData struct {
        Repo        *db.Repository
        CloneURL    string
        IsOwner     bool
        Empty       bool
        Branch      string
        SubPath     string
        ParentPath  string
        Entries     []TreeEntry
        LastCommit  CommitInfo
        Breadcrumbs []Breadcrumb
}


func RepoView(w http.ResponseWriter, r *http.Request, ownerName, repoName, subPath string) {
        viewer := CurrentUser(r)

        repo, err := db.GetRepository(ownerName, repoName)
        if err != nil || repo == nil {
                http.NotFound(w, r)
                return
        }

        if repo.IsPrivate {
                if viewer == nil || viewer.Username != ownerName {
                        http.Error(w, "Repository not found", http.StatusNotFound)
                        return
                }
        }

        data := repoViewData{
                Repo:     repo,
                CloneURL: "http://" + r.Host + "/" + ownerName + "/" + repoName + ".git",
                IsOwner:  viewer != nil && viewer.Username == ownerName,
        }

        dir := repoDir(ownerName, repoName)
        if !gitIsEmpty(dir) {
                branch := gitDefaultBranch(dir)
                data.Empty = false
                data.Branch = branch
                data.SubPath = subPath
                data.Entries = gitListTree(dir, branch, subPath)
                data.LastCommit = gitLastCommit(dir)
                if subPath != "" {
                        idx := strings.LastIndex(subPath, "/")
                        if idx < 0 {
                                data.ParentPath = ""
                        } else {
                                data.ParentPath = "tree/" + subPath[:idx]
                        }
                }
                data.Breadcrumbs = buildBreadcrumbs(ownerName, repoName, subPath)
        } else {
                data.Empty = true
        }

        render(w, "repo.html", PageData{User: viewer, Data: data})
}

// ─── Blob View ─────────────────────────────────────────────────────────────

func BlobView(w http.ResponseWriter, r *http.Request, ownerName, repoName, filePath string) {
        viewer := CurrentUser(r)

        repo, err := db.GetRepository(ownerName, repoName)
        if err != nil || repo == nil {
                http.NotFound(w, r)
                return
        }

        if repo.IsPrivate {
                if viewer == nil || viewer.Username != ownerName {
                        http.Error(w, "Repository not found", http.StatusNotFound)
                        return
                }
        }

        dir := repoDir(ownerName, repoName)
        if gitIsEmpty(dir) {
                http.NotFound(w, r)
                return
        }

        branch := gitDefaultBranch(dir)
        content, isBinary, err := gitFileContent(dir, branch, filePath)
        if err != nil {
                http.NotFound(w, r)
                return
        }

        commit := gitFileLastCommit(dir, branch, filePath)
        fileName := filepath.Base(filePath)

        dirPart := filepath.Dir(filePath)
        var parentPath string
        if dirPart == "." || dirPart == "" {
                parentPath = ""
        } else {
                parentPath = "tree/" + dirPart
        }

        var lines []string
        if !isBinary {
                lines = strings.Split(content, "\n")
                if len(lines) > 0 && lines[len(lines)-1] == "" {
                        lines = lines[:len(lines)-1]
                }
        }

        type blobViewData struct {
                Repo        *db.Repository
                IsOwner     bool
                Branch      string
                FilePath    string
                FileName    string
                ParentPath  string
                Content     string
                Lines       []string
                IsBinary    bool
                LineCount   int
                LastCommit  CommitInfo
                Breadcrumbs []Breadcrumb
        }

        crumbs := []Breadcrumb{{Name: repoName, URL: "/" + ownerName + "/" + repoName}}
        parts := strings.Split(filePath, "/")
        base := "/" + ownerName + "/" + repoName
        for i, part := range parts {
                if i == len(parts)-1 {
                        crumbs = append(crumbs, Breadcrumb{Name: part, URL: ""})
                } else {
                        crumbs = append(crumbs, Breadcrumb{Name: part, URL: base + "/tree/" + strings.Join(parts[:i+1], "/")})
                }
        }

        render(w, "blob.html", PageData{
                User: viewer,
                Data: blobViewData{
                        Repo:        repo,
                        IsOwner:     viewer != nil && viewer.Username == ownerName,
                        Branch:      branch,
                        FilePath:    filePath,
                        FileName:    fileName,
                        ParentPath:  parentPath,
                        Content:     content,
                        Lines:       lines,
                        IsBinary:    isBinary,
                        LineCount:   len(lines),
                        LastCommit:  commit,
                        Breadcrumbs: crumbs,
                },
        })
}

// ─── Git repo init helper ──────────────────────────────────────────────────

func InitGitRepo(owner, name string) error {
        repoPath := filepath.Join("repos", owner, name+".git")
        if err := os.MkdirAll(repoPath, 0755); err != nil {
                return err
        }

        cmd := exec.Command("git", "init", "--bare", repoPath)
        if out, err := cmd.CombinedOutput(); err != nil {
                return fmt.Errorf("git init: %s: %w", string(out), err)
        }
        return nil
}

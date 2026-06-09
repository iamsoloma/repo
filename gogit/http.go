package gogit

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/backend"
)

type Request struct {
	*http.Request
	RepoName string
}

type Server struct {
	AuthFunc func(Credential, *Request) (bool, error)
	backend  *backend.Backend
}

func NewServer(backend *backend.Backend, authFunc func(Credential, *Request) (bool, error)) *Server {
	return &Server{
		AuthFunc: authFunc,
		backend:  backend,
	}
}

var reSlashDedup = regexp.MustCompile(`\/{2,}`)

// Parse out namespace and repository name from the path.
// Examples:
// repo -> "", "repo"
// org/repo -> "org", "repo"
// org/suborg/rpeo -> "org/suborg", "repo"
func getNamespaceAndRepo(input string) (string, string) {
	if input == "" || input == "/" {
		return "", ""
	}

	// Remove duplicate slashes
	input = reSlashDedup.ReplaceAllString(input, "/")

	// Remove leading slash
	if input[0] == '/' && input != "/" {
		input = input[1:]
	}

	blocks := strings.Split(input, "/")
	num := len(blocks)

	if num < 2 {
		return "", blocks[0]
	}

	return strings.Join(blocks[0:num-1], "/"), blocks[num-1]
}

func (s *Server) getRepoURLPath(r *http.Request) string {
	var httpServices = []*regexp.Regexp{
		regexp.MustCompile("(.*?)/HEAD$"),
		regexp.MustCompile("(.*?)/info/refs$"),
		regexp.MustCompile("(.*?)/objects/info/alternates$"),
		regexp.MustCompile("(.*?)/objects/info/http-alternates$"),
		regexp.MustCompile("(.*?)/objects/info/packs$"),
		regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38,62}$"),
		regexp.MustCompile(`(.*?)/objects/pack/pack-[0-9a-f]{40,64}\.pack$`),
		regexp.MustCompile(`(.*?)/objects/pack/pack-[0-9a-f]{40,64}\.idx$`),
		regexp.MustCompile("(.*?)/git-upload-pack$"),
		regexp.MustCompile("(.*?)/git-receive-pack$"),
	}

	for _, re := range httpServices {
		if matches := re.FindStringSubmatch(r.URL.Path); matches != nil {
			repoUrlPath := matches[1]
			return repoUrlPath
		}
	}
	return ""
}

func RepoExists(p string) bool {
	_, err := os.Stat(path.Join(p, "objects"))
	return err == nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s %s %s", r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("Authorization"))

	repoUrlPath := s.getRepoURLPath(r)
	if repoUrlPath == "" {
		log.Printf("invalid request path: %s", r.URL.Path)
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
	repoNamespace, repoName := getNamespaceAndRepo(repoUrlPath)
	if repoName == "" {
		log.Println("auth", fmt.Errorf("no repo name provided"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := &Request{
		Request:  r,
		RepoName: path.Join(repoNamespace, repoName),
	}

	if s.AuthFunc == nil {
		log.Println("auth", fmt.Errorf("no auth backend provided"))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header()["WWW-Authenticate"] = []string{`Basic realm=""`}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	cred, err := getCredential(r)
	if err != nil {
		log.Println("auth", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	allow, err := s.AuthFunc(cred, req)
	if !allow || err != nil {
		if err != nil {
			log.Println("auth", err)
		}

		log.Println("auth", fmt.Errorf("rejected user %s", cred.Username))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Println(repoNamespace+"/"+repoName)
	if !RepoExists(path.Join("./repos", repoNamespace, repoName)) {
		log.Printf("repo does not exist, creating: %s", req.RepoName)
		if err := InitRepo("./repos", req.RepoName); err != nil {
			log.Printf("failed to create repo: %s, error: %v", req.RepoName, err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	s.backend.ServeHTTP(w, r)
}

func InitRepo(base, repo string) error {
	repoPath := filepath.Join(base, repo)
	var err error
	if _, err = os.Stat(repoPath); err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	log.Printf("initializing repo at %s", repoPath)

	if err = os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}

	_, err = git.PlainInit(repoPath, true)
	return err
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"strings"
)

func main() {
	mux := http.NewServeMux()

	// Static file handlers (ensure no method prefix is used here)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./js"))))

	// Route handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.ServeFile(w, r, "templates/form.html")
		} else {
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("POST /encrypt", handleEncryption)
	mux.HandleFunc("POST /api/encrypt", handleAPIEncryption)

	mux.HandleFunc("GET /clear", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/form.html")
	})

	// Start server
	http.ListenAndServe(":8080", mux)
}

func handleEncryption(w http.ResponseWriter, r *http.Request) {
	textToEncrypt := r.FormValue("text")
	encryptedText, err := encryptText(textToEncrypt)
	if err != nil {
		http.Error(w, "Error encrypting text: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderHTML(w, "templates/encrypted_text.html", map[string]string{"EncryptedText": encryptedText})
}

func handleAPIEncryption(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Text string `json:"text"`
	}
	if err := decodeJSONBody(r, &requestData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	encryptedText, err := encryptText(requestData.Text)
	if err != nil {
		http.Error(w, "Error encrypting text", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, encryptedText)
}

func encryptText(text string) (string, error) {
	if isAnsibleVaultAvailable() {
		cmd := exec.Command("ansible-vault", "encrypt_string", text, "--name", "encrypted")
		output, err := cmd.CombinedOutput()
		return string(output), err
	} else {
		return sha256Encrypt(text), nil
	}
}

func isAnsibleVaultAvailable() bool {
	cmd := exec.Command("which", "ansible-vault")
	err := cmd.Run()
	return err == nil
}

func sha256Encrypt(text string) string {
	hash := sha256.New()
	hash.Write([]byte(text))
	hashedText := hex.EncodeToString(hash.Sum(nil))

	// Return output in a format similar to ansible-vault
	return fmt.Sprintf(`$ANSIBLE_VAULT;1.1;SHA256
%s
`, formatHashOutput(hashedText))
}

func formatHashOutput(hashedText string) string {
	var sb strings.Builder
	lineLength := 32
	for i := 0; i < len(hashedText); i += lineLength {
		end := i + lineLength
		if end > len(hashedText) {
			end = len(hashedText)
		}
		sb.WriteString(hashedText[i:end] + "\n")
	}
	return sb.String()
}

func renderHTML(w http.ResponseWriter, templatePath string, data map[string]string) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
	}
}

func decodeJSONBody(r *http.Request, dest interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(dest)
}

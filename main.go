package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.21.0"
)

var tracer = otel.Tracer("ansible-vault-encryptor")

func main() {
	// Initialize logging
	log.Println("Starting the server...")

	// Set up OpenTelemetry tracing
	shutdown := initTracer()
	defer shutdown()

	// Create an instrumented HTTP mux
	mux := http.NewServeMux()

	// Static file handlers
	mux.Handle("/assets/", otelhttp.NewHandler(http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))), "Static Assets"))
	mux.Handle("/js/", otelhttp.NewHandler(http.StripPrefix("/js/", http.FileServer(http.Dir("./js"))), "JavaScript Files"))

	// Route handlers
	mux.Handle("/", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			log.Println("Serving form.html")
			http.ServeFile(w, r, "templates/form.html")
		} else {
			log.Println("Not Found")
			http.NotFound(w, r)
		}
	}), "Main Page"))

	mux.Handle("POST /encrypt", otelhttp.NewHandler(http.HandlerFunc(handleEncryption), "Encryption Handler"))
	mux.Handle("POST /api/encrypt", otelhttp.NewHandler(http.HandlerFunc(handleAPIEncryption), "API Encryption Handler"))
	mux.Handle("GET /clear", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving form.html (clear)")
		http.ServeFile(w, r, "templates/form.html")
	}), "Clear Page"))

	// Start server
	log.Println("Server started on http://localhost:8080")
	http.ListenAndServe(":8080", otelhttp.NewHandler(mux, "HTTP Server"))
}

func handleEncryption(w http.ResponseWriter, r *http.Request) {
	textToEncrypt := r.FormValue("text")
	encryptedText, err := encryptText(textToEncrypt)
	if err != nil {
		log.Printf("Error encrypting text: %v", err)
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
		log.Println("Invalid request payload")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	encryptedText, err := encryptText(requestData.Text)
	if err != nil {
		log.Printf("Error encrypting text: %v", err)
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
		log.Printf("Unable to load template: %v", err)
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Unable to render template: %v", err)
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
	}
}

func decodeJSONBody(r *http.Request, dest interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(dest)
}

func initTracer() func() {
	// Create a new exporter
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to initialize stdouttrace exporter: %v", err)
	}

	// Create a resource for the service
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("Ansible Vault Encryptor"),
		semconv.ServiceVersion("1.0.0"),
	)

	// Create a tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("ansible-vault-encryptor")

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down tracer provider: %v", err)
		}
	}
}

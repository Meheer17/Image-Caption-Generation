package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"
)

type TextRequest struct {
	Text string `json:"text"`
}

type Response struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

var g *genkit.Genkit

// Helper functions to create messages and parts
func NewUserMessage(parts ...*ai.Part) *ai.Message {
	return &ai.Message{
		Role:    ai.RoleUser,
		Content: parts,
	}
}

func NewTextPart(text string) *ai.Part {
	return &ai.Part{
		Text: text,
	}
}

func main() {
	godotenv.Load()

	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_AI_API_KEY environment variable is required")
	}

	var err error
	g, err = genkit.Init(ctx,
		genkit.WithPlugins(
			&googlegenai.GoogleAI{
				APIKey: apiKey,
			},
		),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Setup HTTP routes with CORS middleware
	http.HandleFunc("/", corsMiddleware(serveHTML))
	http.HandleFunc("/health", corsMiddleware(handleHealthCheck))
	http.HandleFunc("/analyze-text", corsMiddleware(handleTextAnalysis))
	http.HandleFunc("/analyze-image", corsMiddleware(handleImageAnalysis))

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// CORS middleware to handle cross-origin requests
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Analysis Tool</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
        }
        .section {
            margin-bottom: 30px;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        .section h2 {
            margin-top: 0;
            color: #555;
        }
        textarea, input[type="file"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        textarea {
            height: 100px;
            resize: vertical;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            margin-top: 10px;
        }
        button:hover {
            background-color: #45a049;
        }
        button:disabled {
            background-color: #cccccc;
            cursor: not-allowed;
        }
        .result {
            margin-top: 15px;
            padding: 15px;
            background-color: #f9f9f9;
            border-left: 4px solid #4CAF50;
            border-radius: 4px;
            white-space: pre-wrap;
        }
        .error {
            border-left-color: #f44336;
            background-color: #ffebee;
            color: #c62828;
        }
        .loading {
            display: none;
            margin-top: 10px;
            color: #666;
        }
        #imagePreview {
            max-width: 100%;
            max-height: 300px;
            margin-top: 10px;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>AI Analysis Tool</h1>

        <div class="section">
            <h2>Text Analysis</h2>
            <textarea id="textInput" placeholder="Enter your text here..."></textarea>
            <button onclick="analyzeText()">Analyze Text</button>
            <div class="loading" id="textLoading">Analyzing text...</div>
            <div id="textResult"></div>
        </div>

        <div class="section">
            <h2>Image Analysis</h2>
            <input type="file" id="imageInput" accept="image/*" onchange="previewImage()">
            <img id="imagePreview" style="display: none;">
            <button onclick="analyzeImage()">Analyze Image</button>
            <div class="loading" id="imageLoading">Analyzing image...</div>
            <div id="imageResult"></div>
        </div>
    </div>

    <script>
        function previewImage() {
            const input = document.getElementById('imageInput');
            const preview = document.getElementById('imagePreview');

            if (input.files && input.files[0]) {
                const reader = new FileReader();
                reader.onload = function(e) {
                    preview.src = e.target.result;
                    preview.style.display = 'block';
                }
                reader.readAsDataURL(input.files[0]);
            }
        }

        async function analyzeText() {
            const text = document.getElementById('textInput').value;
            const resultDiv = document.getElementById('textResult');
            const loadingDiv = document.getElementById('textLoading');

            if (!text.trim()) {
                showResult(resultDiv, 'Please enter some text to analyze.', true);
                return;
            }

            loadingDiv.style.display = 'block';
            resultDiv.innerHTML = '';

            try {
                const response = await fetch('/analyze-text', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ text: text })
                });

                const data = await response.json();
                loadingDiv.style.display = 'none';

                if (data.error) {
                    showResult(resultDiv, data.error, true);
                } else {
                    showResult(resultDiv, data.result, false);
                }
            } catch (error) {
                loadingDiv.style.display = 'none';
                showResult(resultDiv, 'Error: ' + error.message, true);
            }
        }

        async function analyzeImage() {
            const input = document.getElementById('imageInput');
            const resultDiv = document.getElementById('imageResult');
            const loadingDiv = document.getElementById('imageLoading');

            if (!input.files || !input.files[0]) {
                showResult(resultDiv, 'Please select an image to analyze.', true);
                return;
            }

            loadingDiv.style.display = 'block';
            resultDiv.innerHTML = '';

            const formData = new FormData();
            formData.append('image', input.files[0]);

            try {
                const response = await fetch('/analyze-image', {
                    method: 'POST',
                    body: formData
                });

                const data = await response.json();
                loadingDiv.style.display = 'none';

                if (data.error) {
                    showResult(resultDiv, data.error, true);
                } else {
                    showResult(resultDiv, data.result, false);
                }
            } catch (error) {
                loadingDiv.style.display = 'none';
                showResult(resultDiv, 'Error: ' + error.message, true);
            }
        }

        function showResult(div, message, isError) {
            div.innerHTML = '<div class="result' + (isError ? ' error' : '') + '">' + message + '</div>';
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthResponse := map[string]interface{}{
		"status":  "healthy",
		"service": "AI Analysis Tool",
		"version": "1.0.0",
		"message": "Service is running and ready to analyze text and images",
	}

	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		log.Printf("Error encoding health check response: %v", err)
	}
}

func handleTextAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, fmt.Sprintf("Invalid JSON request: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		sendErrorResponse(w, "Text cannot be empty", http.StatusBadRequest)
		return
	}

	systemPrompt := "You are a helpful AI assistant. Analyze and respond to the user's text with insightful, detailed, and helpful information."

	resp, err := genkit.Generate(context.Background(), g,
		ai.WithSystem(systemPrompt),
		ai.WithMessages(
			NewUserMessage(NewTextPart(req.Text)),
		),
	)

	if err != nil {
		log.Printf("Text analysis error: %v", err)
		sendErrorResponse(w, "Failed to analyze text. Please try again later.", http.StatusInternalServerError)
		return
	}

	if resp == nil {
		sendErrorResponse(w, "Received nil response from the model", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, resp.Text())
}

func handleImageAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		log.Printf("Form parsing error: %v", err)
		sendErrorResponse(w, "Error parsing form data. Please ensure you're uploading a valid image file.", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("File retrieval error: %v", err)
		sendErrorResponse(w, "No image file found. Please select an image to upload.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file data
	imageData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("File reading error: %v", err)
		sendErrorResponse(w, "Error reading image file. Please try uploading again.", http.StatusInternalServerError)
		return
	}

	// Validate file size
	if len(imageData) == 0 {
		sendErrorResponse(w, "Empty image file. Please select a valid image.", http.StatusBadRequest)
		return
	}

	// Basic file type validation
	mimeType := getMimeType(header.Filename)
	if mimeType == "application/octet-stream" {
		sendErrorResponse(w, "Unsupported image format. Please upload JPG, PNG, GIF, WebP, or BMP files.", http.StatusBadRequest)
		return
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// Create data URL
	dataUrl := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	systemPrompt := "You are a master of visual image analysis, specialized in providing detailed and insightful captions for images. When analyzing an image, consider:\n\n"
	systemPrompt += "- The main subject(s) and their characteristics\n"
	systemPrompt += "- Background, setting, and context\n"
	systemPrompt += "- Colors, lighting, composition, and mood\n"
	systemPrompt += "- Any notable actions, emotions, or interactions\n"
	systemPrompt += "- Style, artistic elements, and visual storytelling\n\n"
	systemPrompt += "Provide rich, descriptive captions that capture both the literal contents and the essence of the image. Be detailed, specific, and articulate in your analysis."

	resp, err := genkit.Generate(context.Background(), g,
		ai.WithSystem(systemPrompt),
		ai.WithMessages(
			NewUserMessage(
				ai.NewMediaPart(mimeType, dataUrl),
				NewTextPart("Give me a Single Line Caption for the image. which is catchy and interesting to post on so"),
			),
		),
	)

	if err != nil {
		log.Printf("Image analysis error: %v", err)
		sendErrorResponse(w, "Failed to analyze image. Please try again later.", http.StatusInternalServerError)
		return
	}

	if resp == nil {
		sendErrorResponse(w, "Received nil response from the model", http.StatusInternalServerError)
		return
	}

	sendSuccessResponse(w, resp.Text())
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := Response{Error: message}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

func sendSuccessResponse(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := Response{Result: result}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding success response: %v", err)
	}
}

// Helper function to determine MIME type based on file extension
func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

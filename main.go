package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"

	"github.com/firebase/genkit/go/ai"
)

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
	ctx := context.Background()

	g, err := genkit.Init(ctx,
		genkit.WithPlugins(
			&googlegenai.GoogleAI{
				APIKey: "",
			},
		),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Create a system prompt for image analysis
	systemPrompt := "You are a master of visual image analysis, specialized in providing detailed and insightful captions for images. When analyzing an image, consider:\n\n"
	systemPrompt += "- The main subject(s) and their characteristics\n"
	systemPrompt += "- Background, setting, and context\n"
	systemPrompt += "- Colors, lighting, composition, and mood\n"
	systemPrompt += "- Any notable actions, emotions, or interactions\n"
	systemPrompt += "- Style, artistic elements, and visual storytelling\n\n"
	systemPrompt += "Provide rich, descriptive captions that capture both the literal contents and the essence of the image. Be detailed, specific, and articulate in your analysis."

	// Start a forever loop to listen for terminal input
	println("Enter prompts (type END to exit):")
	println("To analyze an image, type 'IMAGE:' followed by the file path")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		print("> ")
		if !scanner.Scan() {
			break
		}

		fullLine := scanner.Text()

		if fullLine == "" {
			continue
		}

		if fullLine == "END" {
			println("Exiting chat...")
			break
		}

		var resp *ai.ModelResponse

		// Check if this is an image request
		if strings.HasPrefix(fullLine, "IMAGE:") {
			imagePath := strings.TrimSpace(strings.TrimPrefix(fullLine, "IMAGE:"))
			println("Processing image:", imagePath)

			// Read the image file
			imageData, err := os.ReadFile(imagePath)
			if err != nil {
				fmt.Printf("Error reading image file: %v\n", err)
				continue
			}

			// Convert image to base64
			base64Data := base64.StdEncoding.EncodeToString(imageData)

			// Print the base64 data (showing first 50 chars)
			if len(base64Data) > 50 {
				fmt.Printf("Base64 data (first 50 chars): %s...\n", base64Data[:50])
			} else {
				fmt.Printf("Base64 data: %s\n", base64Data)
			}

			// Determine MIME type from extension
			mimeType := getMimeType(imagePath)

			// Create data URL
			dataUrl := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
			resp, err = genkit.Generate(ctx, g,
				ai.WithSystem(systemPrompt),
				ai.WithMessages(
					NewUserMessage(
						ai.NewMediaPart(mimeType, dataUrl),
						NewTextPart("Give me a good single line caption for this image."),
					),
				),
			)
		} else {
			// Process regular text prompt
			println("Processing:", fullLine)

			resp, err = genkit.Generate(ctx, g,
				ai.WithSystem(systemPrompt),
			)
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Check if response is nil before using it
		if resp == nil {
			fmt.Println("Received nil response from the model")
			continue
		}

		fmt.Printf("Response: %s\n", resp.Text())
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

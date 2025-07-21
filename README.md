# Image Analysis Tool

A Go application that analyzes images and generates detailed captions using Google Gemini AI.

## Usage

1. Set your Google AI API key in the code
2. Run the application:
   ```
   go run main.go
   ```

3. Enter prompts in the terminal:
   - For text prompts: Just type your message
   - For image analysis: `IMAGE: /path/to/your/image.jpg`
   - To exit: Type `END`

## Supported Image Formats

- JPG/JPEG
- PNG
- GIF
- WebP
- BMP

## Example

```
> IMAGE: ./photo.jpg
Wait For Response
> END
```

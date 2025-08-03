package azure

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "strings"
    "time"
)

// StreamingResponseConverter handles the conversion of Responses API SSE to Chat Completions SSE
type StreamingResponseConverter struct {
    reader io.Reader
    writer io.Writer
    model  string
}

// NewStreamingResponseConverter creates a new streaming converter
func NewStreamingResponseConverter(reader io.Reader, writer io.Writer, model string) *StreamingResponseConverter {
    return &StreamingResponseConverter{
        reader: reader,
        writer: writer,
        model:  model,
    }
}

// Convert performs the streaming conversion
func (c *StreamingResponseConverter) Convert() error {
    scanner := bufio.NewScanner(c.reader)
    var eventType string
    
    for scanner.Scan() {
        line := scanner.Text()
        
        if strings.HasPrefix(line, "event:") {
            eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
            continue
        }
        
        if strings.HasPrefix(line, "data:") {
            data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
            
            switch eventType {
            case "response.output_text.delta":
                c.handleTextDelta(data)
            case "response.completed":
                c.handleCompleted(data)
            case "response.created", "response.in_progress", "response.output_item.added", 
                 "response.output_item.done", "response.content_part.added", 
                 "response.content_part.done", "response.output_text.done":
                // These events don't need to be converted for chat completion streaming
                continue
            }
        }
        
        // Empty line (event separator)
        if line == "" {
            continue
        }
    }
    
    return scanner.Err()
}

func (c *StreamingResponseConverter) handleTextDelta(data string) {
    var deltaEvent map[string]interface{}
    if err := json.Unmarshal([]byte(data), &deltaEvent); err != nil {
        log.Printf("Error parsing delta event: %v", err)
        return
    }
    
    delta, ok := deltaEvent["delta"].(string)
    if !ok {
        return
    }
    
    // Create chat completion chunk
    chunk := map[string]interface{}{
        "id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
        "object":  "chat.completion.chunk",
        "created": time.Now().Unix(),
        "model":   c.model,
        "choices": []map[string]interface{}{
            {
                "index": 0,
                "delta": map[string]interface{}{
                    "content": delta,
                },
                "finish_reason": nil,
            },
        },
    }
    
    c.writeChunk(chunk)
}

func (c *StreamingResponseConverter) handleCompleted(data string) {
    // First send an empty delta to indicate the end of content
    chunk := map[string]interface{}{
        "id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
        "object":  "chat.completion.chunk",
        "created": time.Now().Unix(),
        "model":   c.model,
        "choices": []map[string]interface{}{
            {
                "index": 0,
                "delta": map[string]interface{}{},
                "finish_reason": "stop",
            },
        },
    }
    
    c.writeChunk(chunk)
    
    // Then send the [DONE] marker
    c.writer.Write([]byte("data: [DONE]\n\n"))
    if flusher, ok := c.writer.(flushWriter); ok {
        flusher.Flush()
    }
}

func (c *StreamingResponseConverter) writeChunk(chunk map[string]interface{}) {
    chunkJSON, err := json.Marshal(chunk)
    if err != nil {
        log.Printf("Error marshaling chunk: %v", err)
        return
    }
    
    c.writer.Write([]byte("data: "))
    c.writer.Write(chunkJSON)
    c.writer.Write([]byte("\n\n"))
    
    if flusher, ok := c.writer.(flushWriter); ok {
        flusher.Flush()
    }
}

type flushWriter interface {
    io.Writer
    Flush()
}
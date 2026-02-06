package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// JSONResponse represents a standard response for JSON operations
type JSONResponse struct {
	Success  bool   `json:"success"`
	Data     string `json:"data"`
	Error    string `json:"error"`
	Repaired bool   `json:"repaired"`
}

// ProcessJSON handles the flow: Validate -> Repair (if needed) -> Format
func (a *App) ProcessJSON(input string, indent string, trimWhitespace bool) JSONResponse {
	if input == "" {
		return JSONResponse{Success: true, Data: "", Repaired: false}
	}

	finalJSON := input
	repaired := false

	// 1. Try strict validation first
	if !gjson.Valid(input) {
		// 2. If invalid, try to repair
		repairedText, err := JSONRepair(input, trimWhitespace)
		if err != nil {
			return JSONResponse{
				Success: false,
				Error:   "无法解析 JSON: " + err.Error(),
			}
		}

		// Verify repaired JSON is actually valid
		if !gjson.Valid(repairedText) {
			return JSONResponse{
				Success: false,
				Error:   "修复后的 JSON 仍然无效",
			}
		}
		finalJSON = repairedText
		repaired = true
	}

	// 3. Always Format the result
	var obj interface{}
	if err := json.Unmarshal([]byte(finalJSON), &obj); err != nil {
		return JSONResponse{Success: false, Error: "解析错误: " + err.Error()}
	}

	// 4. If trimWhitespace is requested, trim all strings in the object
	if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	var formatted []byte
	var err error
	if indent == "tab" {
		formatted, err = json.MarshalIndent(obj, "", "\t")
	} else if indent == "2" {
		formatted, err = json.MarshalIndent(obj, "", "  ")
	} else {
		formatted, err = json.MarshalIndent(obj, "", "    ")
	}

	if err != nil {
		return JSONResponse{Success: false, Error: "格式化错误: " + err.Error()}
	}

	return JSONResponse{
		Success:  true,
		Data:     string(formatted),
		Repaired: repaired,
	}
}

// trimStrings recursively trims leading/trailing whitespace from all string values in an interface{}
func (a *App) trimStrings(i interface{}) interface{} {
	switch v := i.(type) {
	case string:
		// Remove leading/trailing whitespace, including \n, \t, etc.
		return strings.Trim(v, " \n\t\r\f\b")
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, val := range v {
			newKey := strings.Trim(k, " \n\t\r\f\b")
			newMap[newKey] = a.trimStrings(val)
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(v))
		for idx, val := range v {
			newSlice[idx] = a.trimStrings(val)
		}
		return newSlice
	default:
		return v
	}
}

// FormatJSON beautifies the JSON string
func (a *App) FormatJSON(input string, indent string, trimWhitespace bool) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		// Try to process first if it's invalid
		resp := a.ProcessJSON(input, indent, trimWhitespace)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	var formatted []byte
	if indent == "tab" {
		formatted, err = json.MarshalIndent(obj, "", "\t")
	} else if indent == "2" {
		formatted, err = json.MarshalIndent(obj, "", "  ")
	} else {
		formatted, err = json.MarshalIndent(obj, "", "    ")
	}

	if err != nil {
		return JSONResponse{Success: false, Error: err.Error()}
	}

	return JSONResponse{Success: true, Data: string(formatted)}
}

// MinifyJSON removes all whitespace
func (a *App) MinifyJSON(input string, trimWhitespace bool) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	minified, err := json.Marshal(obj)
	if err != nil {
		return JSONResponse{Success: false, Error: err.Error()}
	}

	return JSONResponse{Success: true, Data: string(minified)}
}

// ConvertToYAML converts JSON to YAML
func (a *App) ConvertToYAML(input string, trimWhitespace bool) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	yamlData, err := yaml.Marshal(obj)
	if err != nil {
		return JSONResponse{Success: false, Error: err.Error()}
	}

	return JSONResponse{Success: true, Data: string(yamlData)}
}

// GetPathByOffset returns the JSON path for a given character offset
func (a *App) GetPathByOffset(input string, offset int) string {
	if input == "" {
		return "$"
	}

	// 转换为 rune 偏移，以匹配 Monaco
	runes := []rune(input)
	if offset < 0 || offset > len(runes) {
		return "$"
	}

	// 将 rune 偏移转回字节偏移
	byteOffset := len(string(runes[:offset]))

	res := gjson.Parse(input)
	// 如果是空对象或空数组，或者解析失败
	if !res.Exists() || (res.Raw == "" && input != "") {
		return "$"
	}

	return findPathRecursive(res, byteOffset, "$")
}

func findPathRecursive(res gjson.Result, byteOffset int, currentPath string) string {
	if !res.IsObject() && !res.IsArray() {
		return currentPath
	}

	found := false
	var nextPath string
	var foundVal gjson.Result

	if res.IsObject() {
		res.ForEach(func(key, value gjson.Result) bool {
			// 检查是否点击了键
			keyStart := key.Index
			keyEnd := key.Index + len(key.Raw)
			if keyStart <= byteOffset && byteOffset < keyEnd {
				nextPath = currentPath + "." + key.String()
				foundVal = value
				found = true
				return false
			}
			// 检查是否点击了值区域
			valStart := value.Index
			valEnd := value.Index + len(value.Raw)
			if valStart <= byteOffset && byteOffset < valEnd {
				nextPath = currentPath + "." + key.String()
				foundVal = value
				found = true
				return false
			}
			return true
		})
	} else if res.IsArray() {
		idx := 0
		res.ForEach(func(key, value gjson.Result) bool {
			valStart := value.Index
			valEnd := value.Index + len(value.Raw)
			if valStart <= byteOffset && byteOffset < valEnd {
				nextPath = fmt.Sprintf("%s[%d]", currentPath, idx)
				foundVal = value
				found = true
				return false
			}
			idx++
			return true
		})
	}

	if found {
		// 递归查找更深层
		if foundVal.IsObject() || foundVal.IsArray() {
			return findPathRecursive(foundVal, byteOffset, nextPath)
		}
		return nextPath
	}

	// 如果没有找到匹配的子节点，且不是根节点，说明点击了空白处
	// 为了避免误定位到父节点，返回空字符串
	return ""
}

type PathInfo struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

// GetPathOffset returns the character offset and length of a JSON path in the input string
func (a *App) GetPathOffset(input string, path string) PathInfo {
	if path == "" || path == "$" {
		return PathInfo{Offset: 0, Length: 1}
	}

	// Normalize path for gjson: $.store.book[0] -> store.book.0
	searchPath := path
	if strings.HasPrefix(searchPath, "$.") {
		searchPath = searchPath[2:]
	} else if strings.HasPrefix(searchPath, "$") {
		searchPath = searchPath[1:]
	}

	// Replace [n] with .n
	searchPath = strings.ReplaceAll(searchPath, "[", ".")
	searchPath = strings.ReplaceAll(searchPath, "]", "")
	if strings.HasPrefix(searchPath, ".") {
		searchPath = searchPath[1:]
	}

	res := gjson.Get(input, searchPath)
	if res.Exists() {
		offset := res.Index
		length := len(res.Raw)

		// 如果路径包含点（说明不是根级成员），尝试向上找 Key 的位置
		// 逻辑：在 res.Index 之前寻找最近的一个引号包裹的 Key
		if strings.Contains(path, ".") {
			parts := strings.Split(path, ".")
			lastKey := parts[len(parts)-1]
			// 如果 key 包含数组索引如 [0]，提取出真正的 key 部分
			if idx := strings.Index(lastKey, "["); idx != -1 {
				lastKey = lastKey[:idx]
			}

			if lastKey != "" && lastKey != "$" {
				// 在 offset 之前寻找 lastKey
				searchArea := input[:offset]
				keyPattern := "\"" + lastKey + "\""
				keyIdx := strings.LastIndex(searchArea, keyPattern)
				if keyIdx != -1 {
					// 找到了 Key，将 offset 提前到 Key 的起始位置
					// 并重新计算长度，覆盖从 Key 到 Value 的结束
					length = (offset + length) - keyIdx
					offset = keyIdx
				}
			}
		}

		// Convert byte index to character index (rune index) for Monaco
		runesBefore := []rune(input[:offset])
		// 注意：这里的 length 是字节长度，需要正确计算字符长度
		contentToHighlight := input[offset : offset+length]
		highlightRunes := []rune(contentToHighlight)

		return PathInfo{
			Offset: len(runesBefore),
			Length: len(highlightRunes),
		}
	}

	return PathInfo{Offset: -1, Length: 0}
}

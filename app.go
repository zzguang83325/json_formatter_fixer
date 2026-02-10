package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
func (a *App) ProcessJSON(input string, indent string, trimWhitespace bool, keepOrder bool) JSONResponse {
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

	// 3. Format the result
	var formatted []byte
	var err error

	if trimWhitespace && keepOrder {
		// Special case: both enabled. Use gjson to traverse and reconstruct while trimming
		// to preserve original key order.
		trimmedCompact := a.reconstructAndTrim(gjson.Parse(finalJSON))
		finalJSON = trimmedCompact
	}

	if !keepOrder {
		// If we don't need to keep order, standard json package sorts keys alphabetically
		var obj interface{}
		if err := json.Unmarshal([]byte(finalJSON), &obj); err != nil {
			return JSONResponse{Success: false, Error: "解析错误: " + err.Error()}
		}

		if trimWhitespace {
			obj = a.trimStrings(obj)
		}

		if indent == "tab" {
			formatted, err = json.MarshalIndent(obj, "", "\t")
		} else if indent == "2" {
			formatted, err = json.MarshalIndent(obj, "", "  ")
		} else {
			formatted, err = json.MarshalIndent(obj, "", "    ")
		}
	} else {
		// We want to keep order. If trimWhitespace was handled above, finalJSON is already trimmed.
		var indentStr string
		if indent == "tab" {
			indentStr = "\t"
		} else if indent == "2" {
			indentStr = "  "
		} else {
			indentStr = "    "
		}

		var buf bytes.Buffer
		err = json.Indent(&buf, []byte(finalJSON), "", indentStr)
		if err != nil {
			// Fallback to pretty.Pretty if json.Indent fails
			formatted = pretty.PrettyOptions([]byte(finalJSON), &pretty.Options{
				Indent:   indentStr,
				SortKeys: false, // Maintain original order
			})
			err = nil
		} else {
			formatted = buf.Bytes()
		}
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

// reconstructAndTrim recursively traverses a gjson.Result and reconstructs the JSON string
// while trimming strings and preserving original key order.
func (a *App) reconstructAndTrim(res gjson.Result) string {
	if res.IsArray() {
		var sb strings.Builder
		sb.WriteString("[")
		first := true
		res.ForEach(func(key, value gjson.Result) bool {
			if !first {
				sb.WriteString(",")
			}
			sb.WriteString(a.reconstructAndTrim(value))
			first = false
			return true
		})
		sb.WriteString("]")
		return sb.String()
	}
	if res.IsObject() {
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		res.ForEach(func(key, value gjson.Result) bool {
			if !first {
				sb.WriteString(",")
			}
			// Trim key
			trimmedKey := strings.Trim(key.String(), " \n\t\r\f\b")
			sb.WriteString(fmt.Sprintf("%q:", trimmedKey))
			sb.WriteString(a.reconstructAndTrim(value))
			first = false
			return true
		})
		sb.WriteString("}")
		return sb.String()
	}
	if res.Type == gjson.String {
		return fmt.Sprintf("%q", strings.Trim(res.String(), " \n\t\r\f\b"))
	}
	// For other types (Number, True, False, Null), use Raw
	return res.Raw
}

// FormatJSON beautifies the JSON string
func (a *App) FormatJSON(input string, indent string, trimWhitespace bool, keepOrder bool) JSONResponse {
	// If it's invalid or we need to trim whitespace or sort keys, use ProcessJSON which handles these cases
	if !gjson.Valid(input) || trimWhitespace || !keepOrder {
		return a.ProcessJSON(input, indent, trimWhitespace, keepOrder)
	}

	// For valid JSON without trimming and keeping order, use json.Indent to preserve order
	var indentStr string
	if indent == "tab" {
		indentStr = "\t"
	} else if indent == "2" {
		indentStr = "  "
	} else {
		indentStr = "    "
	}

	var buf bytes.Buffer
	err := json.Indent(&buf, []byte(input), "", indentStr)
	if err != nil {
		return JSONResponse{Success: false, Error: err.Error()}
	}

	return JSONResponse{Success: true, Data: buf.String()}
}

// MinifyJSON removes all whitespace
func (a *App) MinifyJSON(input string, trimWhitespace bool, keepOrder bool) JSONResponse {
	finalJSON := input
	if !gjson.Valid(input) {
		resp := a.ProcessJSON(input, "0", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		finalJSON = resp.Data
	}

	if trimWhitespace && keepOrder {
		// Use reconstructAndTrim to preserve order while trimming
		finalJSON = a.reconstructAndTrim(gjson.Parse(finalJSON))
	}

	if !keepOrder {
		var obj interface{}
		if err := json.Unmarshal([]byte(finalJSON), &obj); err != nil {
			return JSONResponse{Success: false, Error: err.Error()}
		}

		if trimWhitespace {
			obj = a.trimStrings(obj)
		}

		minified, err := json.Marshal(obj)
		if err != nil {
			return JSONResponse{Success: false, Error: err.Error()}
		}
		return JSONResponse{Success: true, Data: string(minified)}
	}

	var buf bytes.Buffer
	err := json.Compact(&buf, []byte(finalJSON))
	if err != nil {
		return JSONResponse{Success: false, Error: err.Error()}
	}

	return JSONResponse{Success: true, Data: buf.String()}
}

// ConvertToYAML converts JSON to YAML
func (a *App) ConvertToYAML(input string, trimWhitespace bool, keepOrder bool) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
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

// ConvertToJavaClass converts JSON to Java class
func (a *App) ConvertToJavaClass(input string, trimWhitespace bool, keepOrder bool, className string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if className == "" {
		className = "RootClass"
	}

	javaCode := a.generateJavaClass(className, obj)
	return JSONResponse{Success: true, Data: javaCode}
}

// generateJavaClass generates Java class code from interface{}
func (a *App) generateJavaClass(className string, obj interface{}) string {
	var builder strings.Builder
	classes := make(map[string]string)

	a.collectJavaClasses(className, obj, classes)

	for _, classDef := range classes {
		builder.WriteString(classDef)
		builder.WriteString("\n")
	}

	return builder.String()
}

// collectJavaClasses recursively collects all Java class definitions
func (a *App) collectJavaClasses(className string, obj interface{}, classes map[string]string) {
	if _, exists := classes[className]; exists {
		return
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		var builder strings.Builder
		builder.WriteString("import java.util.*;\n\n")
		builder.WriteString("public class ")
		builder.WriteString(className)
		builder.WriteString(" {\n")

		for key, value := range v {
			fieldName := toCamelCase(key)
			javaType := a.getJavaType(value, className, fieldName)
			builder.WriteString("    private ")
			builder.WriteString(javaType)
			builder.WriteString(" ")
			builder.WriteString(fieldName)
			builder.WriteString(";\n")

			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedClassName := strings.ToUpper(fieldName[:1]) + fieldName[1:]
				a.collectJavaClasses(nestedClassName, nestedMap, classes)
			} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
				if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
					nestedClassName := strings.ToUpper(fieldName[:1]) + fieldName[1:]
					a.collectJavaClasses(nestedClassName, nestedMap, classes)
				}
			}
		}

		builder.WriteString("\n")
		for key := range v {
			fieldName := toCamelCase(key)
			capitalized := strings.ToUpper(fieldName[:1]) + fieldName[1:]

			builder.WriteString("    public ")
			builder.WriteString(capitalized)
			builder.WriteString(" get")
			builder.WriteString(capitalized)
			builder.WriteString("() {\n")
			builder.WriteString("        return this.")
			builder.WriteString(fieldName)
			builder.WriteString(";\n")
			builder.WriteString("    }\n\n")

			builder.WriteString("    public void set")
			builder.WriteString(capitalized)
			builder.WriteString("(")
			builder.WriteString(a.getJavaType(v[key], className, fieldName))
			builder.WriteString(" ")
			builder.WriteString(fieldName)
			builder.WriteString(") {\n")
			builder.WriteString("        this.")
			builder.WriteString(fieldName)
			builder.WriteString(" = ")
			builder.WriteString(fieldName)
			builder.WriteString(";\n")
			builder.WriteString("    }\n\n")
		}

		builder.WriteString("}")
		classes[className] = builder.String()

	case []interface{}:
		if len(v) > 0 {
			a.collectJavaClasses(className, v[0], classes)
		}
	}
}

// getJavaType returns Java type for a value
func (a *App) getJavaType(value interface{}, className string, fieldName string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return "Integer"
		}
		return "Double"
	case bool:
		return "Boolean"
	case string:
		return "String"
	case map[string]interface{}:
		return strings.ToUpper(fieldName[:1]) + fieldName[1:]
	case []interface{}:
		if len(v) > 0 {
			elemType := a.getJavaType(v[0], className, fieldName)
			return "List<" + elemType + ">"
		}
		return "List<Object>"
	case nil:
		return "Object"
	default:
		return "Object"
	}
}

// ConvertToGoStruct converts JSON to Go struct
func (a *App) ConvertToGoStruct(input string, trimWhitespace bool, keepOrder bool, structName string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if structName == "" {
		structName = "RootStruct"
	}

	goCode := a.generateGoStruct(structName, obj)
	return JSONResponse{Success: true, Data: goCode}
}

// generateGoStruct generates Go struct code from interface{}
func (a *App) generateGoStruct(structName string, obj interface{}) string {
	var builder strings.Builder
	structs := make(map[string]string)

	a.collectGoStructs(structName, obj, structs)

	for _, structDef := range structs {
		builder.WriteString(structDef)
		builder.WriteString("\n")
	}

	return builder.String()
}

// collectGoStructs recursively collects all Go struct definitions
func (a *App) collectGoStructs(structName string, obj interface{}, structs map[string]string) {
	if _, exists := structs[structName]; exists {
		return
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		var builder strings.Builder
		builder.WriteString("type ")
		builder.WriteString(structName)
		builder.WriteString(" struct {\n")

		for key, value := range v {
			fieldName := toPascalCase(key)
			goType := a.getGoType(value, structName, fieldName)
			builder.WriteString("    ")
			builder.WriteString(fieldName)
			builder.WriteString(" ")
			builder.WriteString(goType)
			builder.WriteString(" `json:\"")
			builder.WriteString(key)
			builder.WriteString("\"`\n")

			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedStructName := fieldName
				a.collectGoStructs(nestedStructName, nestedMap, structs)
			} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
				if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
					nestedStructName := fieldName
					a.collectGoStructs(nestedStructName, nestedMap, structs)
				}
			}
		}

		builder.WriteString("}")
		structs[structName] = builder.String()

	case []interface{}:
		if len(v) > 0 {
			a.collectGoStructs(structName, v[0], structs)
		}
	}
}

// getGoType returns Go type for a value
func (a *App) getGoType(value interface{}, structName string, fieldName string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return "int"
		}
		return "float64"
	case bool:
		return "bool"
	case string:
		return "string"
	case map[string]interface{}:
		return fieldName
	case []interface{}:
		if len(v) > 0 {
			elemType := a.getGoType(v[0], structName, fieldName)
			return "[]" + elemType
		}
		return "[]interface{}"
	case nil:
		return "interface{}"
	default:
		return "interface{}"
	}
}

// ConvertToPythonClass converts JSON to Python class
func (a *App) ConvertToPythonClass(input string, trimWhitespace bool, keepOrder bool, className string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if className == "" {
		className = "RootClass"
	}

	pythonCode := a.generatePythonClass(className, obj)
	return JSONResponse{Success: true, Data: pythonCode}
}

// generatePythonClass generates Python class code from interface{}
func (a *App) generatePythonClass(className string, obj interface{}) string {
	var builder strings.Builder
	classes := make(map[string]string)

	a.collectPythonClasses(className, obj, classes)

	for _, classDef := range classes {
		builder.WriteString(classDef)
		builder.WriteString("\n")
	}

	return builder.String()
}

// collectPythonClasses recursively collects all Python class definitions
func (a *App) collectPythonClasses(className string, obj interface{}, classes map[string]string) {
	if _, exists := classes[className]; exists {
		return
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		var builder strings.Builder
		builder.WriteString("class ")
		builder.WriteString(className)
		builder.WriteString(":\n")

		for key, value := range v {
			fieldName := toSnakeCase(key)
			pythonType := a.getPythonType(value, className, fieldName)
			builder.WriteString("    ")
			builder.WriteString(fieldName)
			builder.WriteString(": ")
			builder.WriteString(pythonType)
			builder.WriteString("\n")

			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedClassName := toPascalCase(key)
				a.collectPythonClasses(nestedClassName, nestedMap, classes)
			} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
				if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
					nestedClassName := toPascalCase(key)
					a.collectPythonClasses(nestedClassName, nestedMap, classes)
				}
			}
		}

		classes[className] = builder.String()

	case []interface{}:
		if len(v) > 0 {
			a.collectPythonClasses(className, v[0], classes)
		}
	}
}

// getPythonType returns Python type for a value
func (a *App) getPythonType(value interface{}, className string, fieldName string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return "int"
		}
		return "float"
	case bool:
		return "bool"
	case string:
		return "str"
	case map[string]interface{}:
		return toPascalCase(fieldName)
	case []interface{}:
		if len(v) > 0 {
			elemType := a.getPythonType(v[0], className, fieldName)
			return "list[" + elemType + "]"
		}
		return "list"
	case nil:
		return "Optional[Any] = None"
	default:
		return "Any"
	}
}

// toSnakeCase converts camelCase or PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// ConvertToTypeScriptInterface converts JSON to TypeScript interface
func (a *App) ConvertToTypeScriptInterface(input string, trimWhitespace bool, keepOrder bool, interfaceName string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if interfaceName == "" {
		interfaceName = "RootInterface"
	}

	tsCode := a.generateTypeScriptInterface(interfaceName, obj)
	return JSONResponse{Success: true, Data: tsCode}
}

// generateTypeScriptInterface generates TypeScript interface code from interface{}
func (a *App) generateTypeScriptInterface(interfaceName string, obj interface{}) string {
	var builder strings.Builder
	interfaces := make(map[string]string)

	a.collectTypeScriptInterfaces(interfaceName, obj, interfaces)

	for _, interfaceDef := range interfaces {
		builder.WriteString(interfaceDef)
		builder.WriteString("\n")
	}

	return builder.String()
}

// collectTypeScriptInterfaces recursively collects all TypeScript interface definitions
func (a *App) collectTypeScriptInterfaces(interfaceName string, obj interface{}, interfaces map[string]string) {
	if _, exists := interfaces[interfaceName]; exists {
		return
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		var builder strings.Builder
		builder.WriteString("export interface ")
		builder.WriteString(interfaceName)
		builder.WriteString(" {\n")

		for key, value := range v {
			fieldName := toCamelCase(key)
			tsType := a.getTypeScriptType(value, interfaceName, fieldName)
			builder.WriteString("    ")
			builder.WriteString(fieldName)
			builder.WriteString(": ")
			builder.WriteString(tsType)
			builder.WriteString(";\n")

			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedInterfaceName := strings.ToUpper(fieldName[:1]) + fieldName[1:]
				a.collectTypeScriptInterfaces(nestedInterfaceName, nestedMap, interfaces)
			} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
				if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
					nestedInterfaceName := strings.ToUpper(fieldName[:1]) + fieldName[1:]
					a.collectTypeScriptInterfaces(nestedInterfaceName, nestedMap, interfaces)
				}
			}
		}

		builder.WriteString("}")
		interfaces[interfaceName] = builder.String()

	case []interface{}:
		if len(v) > 0 {
			a.collectTypeScriptInterfaces(interfaceName, v[0], interfaces)
		}
	}
}

// getTypeScriptType returns TypeScript type for a value
func (a *App) getTypeScriptType(value interface{}, interfaceName string, fieldName string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return "number"
		}
		return "number"
	case bool:
		return "boolean"
	case string:
		return "string"
	case map[string]interface{}:
		return strings.ToUpper(fieldName[:1]) + fieldName[1:]
	case []interface{}:
		if len(v) > 0 {
			elemType := a.getTypeScriptType(v[0], interfaceName, fieldName)
			return elemType + "[]"
		}
		return "any[]"
	case nil:
		return "any | null"
	default:
		return "any"
	}
}

// ConvertToCSharpClass converts JSON to C# class
func (a *App) ConvertToCSharpClass(input string, trimWhitespace bool, keepOrder bool, className string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if className == "" {
		className = "RootClass"
	}

	csharpCode := a.generateCSharpClass(className, obj)
	return JSONResponse{Success: true, Data: csharpCode}
}

// generateCSharpClass generates C# class code from interface{}
func (a *App) generateCSharpClass(className string, obj interface{}) string {
	var builder strings.Builder
	classes := make(map[string]string)

	a.collectCSharpClasses(className, obj, classes)

	for _, classDef := range classes {
		builder.WriteString(classDef)
		builder.WriteString("\n")
	}

	return builder.String()
}

// collectCSharpClasses recursively collects all C# class definitions
func (a *App) collectCSharpClasses(className string, obj interface{}, classes map[string]string) {
	if _, exists := classes[className]; exists {
		return
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		var builder strings.Builder
		builder.WriteString("public class ")
		builder.WriteString(className)
		builder.WriteString("\n{\n")

		for key, value := range v {
			fieldName := toPascalCase(key)
			csharpType := a.getCSharpType(value, className, fieldName)
			builder.WriteString("    public ")
			builder.WriteString(csharpType)
			builder.WriteString(" ")
			builder.WriteString(fieldName)
			builder.WriteString(" { get; set; }\n")

			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedClassName := fieldName
				a.collectCSharpClasses(nestedClassName, nestedMap, classes)
			} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
				if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
					nestedClassName := fieldName
					a.collectCSharpClasses(nestedClassName, nestedMap, classes)
				}
			}
		}

		builder.WriteString("}")
		classes[className] = builder.String()

	case []interface{}:
		if len(v) > 0 {
			a.collectCSharpClasses(className, v[0], classes)
		}
	}
}

// getCSharpType returns C# type for a value
func (a *App) getCSharpType(value interface{}, className string, fieldName string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return "int"
		}
		return "double"
	case bool:
		return "bool"
	case string:
		return "string"
	case map[string]interface{}:
		return fieldName
	case []interface{}:
		if len(v) > 0 {
			elemType := a.getCSharpType(v[0], className, fieldName)
			return "List<" + elemType + ">"
		}
		return "List<object>"
	case nil:
		return "object"
	default:
		return "object"
	}
}

// toCamelCase converts snake_case or kebab-case to camelCase
func toCamelCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	result := words[0]
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(words[i][:1]) + words[i][1:]
		}
	}
	return result
}

// toPascalCase converts snake_case or kebab-case to PascalCase
func toPascalCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var result string
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return result
}

// ConvertToSQL converts JSON to SQL CREATE TABLE statement
func (a *App) ConvertToSQL(input string, trimWhitespace bool, keepOrder bool, databaseType string, tableName string) JSONResponse {
	var obj interface{}
	err := json.Unmarshal([]byte(input), &obj)
	if err != nil {
		resp := a.ProcessJSON(input, "4", trimWhitespace, keepOrder)
		if !resp.Success {
			return resp
		}
		json.Unmarshal([]byte(resp.Data), &obj)
	} else if trimWhitespace {
		obj = a.trimStrings(obj)
	}

	if tableName == "" {
		tableName = "table1"
	}

	sqlCode := a.generateSQL(obj, databaseType, tableName)
	return JSONResponse{Success: true, Data: sqlCode}
}

// generateSQL generates SQL CREATE TABLE statement from interface{}
func (a *App) generateSQL(obj interface{}, databaseType string, tableName string) string {
	switch v := obj.(type) {
	case map[string]interface{}:
		return a.generateSQLFromMap(v, databaseType, tableName, make(map[string]bool))
	case []interface{}:
		if len(v) > 0 {
			return a.generateSQL(v[0], databaseType, tableName)
		}
		return "-- No data to convert"
	default:
		return "-- Invalid JSON structure for SQL conversion"
	}
}

// generateSQLFromMap generates SQL from map[string]interface{}
func (a *App) generateSQLFromMap(data map[string]interface{}, databaseType string, tableName string, generatedTables map[string]bool) string {
	var builder strings.Builder

	builder.WriteString("-- ")
	builder.WriteString(databaseType)
	builder.WriteString(" CREATE TABLE statement\n")
	builder.WriteString("CREATE TABLE ")
	builder.WriteString(tableName)
	builder.WriteString(" (\n")

	var columns []string
	for key, value := range data {
		columnName := toSnakeCase(key)
		columnType := a.getSQLType(value, databaseType)
		columns = append(columns, "    "+columnName+" "+columnType)
	}

	builder.WriteString(strings.Join(columns, ",\n"))
	builder.WriteString("\n)")

	switch databaseType {
	case "mysql":
		builder.WriteString(" ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
	case "postgresql":
		builder.WriteString(";")
	case "sqlite":
		builder.WriteString(";")
	case "sqlserver":
		builder.WriteString(";")
	case "oracle":
		builder.WriteString(";")
	default:
		builder.WriteString(";")
	}

	builder.WriteString("\n\n")

	generatedTables[tableName] = true

	for key, value := range data {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			nestedTableName := toSnakeCase(key)
			if !generatedTables[nestedTableName] {
				builder.WriteString(a.generateSQLFromMap(nestedMap, databaseType, nestedTableName, generatedTables))
			}
		} else if nestedArray, ok := value.([]interface{}); ok && len(nestedArray) > 0 {
			if nestedMap, ok := nestedArray[0].(map[string]interface{}); ok {
				nestedTableName := toSnakeCase(key)
				if !generatedTables[nestedTableName] {
					builder.WriteString(a.generateSQLFromMap(nestedMap, databaseType, nestedTableName, generatedTables))
				}
			}
		}
	}

	return builder.String()
}

// getSQLType returns SQL type for a value based on database type
func (a *App) getSQLType(value interface{}, databaseType string) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			switch databaseType {
			case "mysql":
				return "BIGINT"
			case "postgresql":
				return "INT8"
			case "sqlite":
				return "INTEGER"
			case "sqlserver":
				return "BIGINT"
			case "oracle":
				return "NUMBER(19)"
			default:
				return "BIGINT"
			}
		}
		switch databaseType {
		case "mysql":
			return "DECIMAL(20,10)"
		case "postgresql":
			return "NUMERIC(20,10)"
		case "sqlite":
			return "REAL"
		case "sqlserver":
			return "NUMERIC(20,10)"
		case "oracle":
			return "NUMBER(20,10)"
		default:
			return "DECIMAL(20,10)"
		}
	case bool:
		switch databaseType {
		case "mysql":
			return "TINYINT(1)"
		case "postgresql":
			return "BOOLEAN"
		case "sqlite":
			return "INTEGER"
		case "sqlserver":
			return "BIT"
		case "oracle":
			return "NUMBER(1)"
		default:
			return "SMALLINT"
		}
	case string:
		switch databaseType {
		case "mysql":
			return "VARCHAR(255)"
		case "postgresql":
			return "TEXT"
		case "sqlite":
			return "VARCHAR(255)"
		case "sqlserver":
			return "NVARCHAR(255)"
		case "oracle":
			return "VARCHAR2(255)"
		default:
			return "VARCHAR(255)"
		}
	case map[string]interface{}:
		switch databaseType {
		case "mysql":
			return "JSON"
		case "postgresql":
			return "JSONB"
		case "sqlite":
			return "TEXT"
		case "sqlserver":
			return "NVARCHAR(MAX)"
		case "oracle":
			return "CLOB"
		default:
			return "TEXT"
		}
	case []interface{}:
		switch databaseType {
		case "mysql":
			return "JSON"
		case "postgresql":
			return "JSONB"
		case "sqlite":
			return "TEXT"
		case "sqlserver":
			return "NVARCHAR(MAX)"
		case "oracle":
			return "CLOB"
		default:
			return "TEXT"
		}
	case nil:
		switch databaseType {
		case "mysql":
			return "VARCHAR(255)"
		case "postgresql":
			return "TEXT"
		case "sqlite":
			return "VARCHAR(255)"
		case "sqlserver":
			return "NVARCHAR(255)"
		case "oracle":
			return "VARCHAR2(255)"
		default:
			return "VARCHAR(255)"
		}
	default:
		return "TEXT"
	}
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

// SaveFile saves content to a file, opening a dialog if filename is empty
func (a *App) SaveFile(content string, defaultFilename string) JSONResponse {
	var targetPath string
	var err error

	if defaultFilename == "" {
		// Open save dialog
		targetPath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			DefaultFilename: "data.json",
			Title:           "保存 JSON 文件",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
				{DisplayName: "All Files (*.*)", Pattern: "*.*"},
			},
		})
		if err != nil {
			return JSONResponse{Success: false, Error: "打开保存对话框失败: " + err.Error()}
		}
		if targetPath == "" {
			return JSONResponse{Success: false, Error: "用户取消保存"}
		}
	} else {
		// Use provided filename (if it's a full path, use it; otherwise open dialog with it)
		if strings.Contains(defaultFilename, string(os.PathSeparator)) || strings.Contains(defaultFilename, "/") {
			targetPath = defaultFilename
		} else {
			targetPath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
				DefaultFilename: defaultFilename,
				Title:           "保存文件",
				Filters: []runtime.FileFilter{
					{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
					{DisplayName: "All Files (*.*)", Pattern: "*.*"},
				},
			})
			if err != nil {
				return JSONResponse{Success: false, Error: "打开保存对话框失败: " + err.Error()}
			}
			if targetPath == "" {
				return JSONResponse{Success: false, Error: "用户取消保存"}
			}
		}
	}

	// Write to file
	err = os.WriteFile(targetPath, []byte(content), 0644)
	if err != nil {
		return JSONResponse{Success: false, Error: "写入文件失败: " + err.Error()}
	}

	return JSONResponse{Success: true, Data: targetPath}
}

// WriteFileDirect writes content directly to a specified path without opening a dialog
func (a *App) WriteFileDirect(content string, filePath string) JSONResponse {
	if filePath == "" {
		return JSONResponse{Success: false, Error: "文件路径不能为空"}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return JSONResponse{Success: false, Error: "写入文件失败: " + err.Error()}
	}

	return JSONResponse{Success: true, Data: filePath}
}

// ReadFile reads content from a specified path
func (a *App) ReadFile(filePath string) JSONResponse {
	if filePath == "" {
		return JSONResponse{Success: false, Error: "文件路径不能为空"}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return JSONResponse{Success: false, Error: "读取文件失败: " + err.Error()}
	}

	return JSONResponse{Success: true, Data: string(content)}
}

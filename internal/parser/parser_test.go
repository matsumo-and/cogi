package parser

import (
	"context"
	"testing"
)

// TestGoParser tests Go language parsing
func TestGoParser(t *testing.T) {
	sourceCode := []byte(`
package main

import "fmt"

type User struct {
	Name string
}

func NewUser(name string) *User {
	return &User{Name: name}
}

func (u *User) Greet() {
	fmt.Println("Hello, " + u.Name)
}

func main() {
	user := NewUser("Alice")
	user.Greet()
}
`)

	p, err := New(LangGo)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	result, err := p.Parse(context.Background(), sourceCode)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Check symbols
	if len(result.Symbols) < 4 {
		t.Errorf("Expected at least 4 symbols, got %d", len(result.Symbols))
	}

	// Check for User struct
	foundStruct := false
	for _, sym := range result.Symbols {
		if sym.Name == "User" && sym.Kind == "struct" {
			foundStruct = true
			break
		}
	}
	if !foundStruct {
		t.Error("User struct not found")
	}

	// Check imports
	if len(result.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(result.Imports))
	}
	if len(result.Imports) > 0 && result.Imports[0].ImportPath != "fmt" {
		t.Errorf("Expected import 'fmt', got '%s'", result.Imports[0].ImportPath)
	}

	// Check call sites
	if len(result.CallSites) < 2 {
		t.Errorf("Expected at least 2 call sites, got %d", len(result.CallSites))
	}

	// Check for NewUser call
	foundCall := false
	for _, call := range result.CallSites {
		if call.CalleeName == "NewUser" {
			foundCall = true
			if call.CallType != "direct" {
				t.Errorf("NewUser call should be 'direct', got '%s'", call.CallType)
			}
			break
		}
	}
	if !foundCall {
		t.Error("NewUser call not found")
	}
}

// TestTypeScriptParser tests TypeScript language parsing
func TestTypeScriptParser(t *testing.T) {
	sourceCode := []byte(`
import { Logger } from './logger';

export class User {
	constructor(private name: string) {}

	greet(): void {
		Logger.log("Hello, " + this.name);
	}
}

export function createUser(name: string): User {
	return new User(name);
}
`)

	p, err := New(LangTypeScript)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	result, err := p.Parse(context.Background(), sourceCode)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Check symbols
	if len(result.Symbols) < 2 {
		t.Errorf("Expected at least 2 symbols, got %d", len(result.Symbols))
	}

	// Check for User class
	foundClass := false
	for _, sym := range result.Symbols {
		if sym.Name == "User" && sym.Kind == "class" {
			foundClass = true
			break
		}
	}
	if !foundClass {
		t.Error("User class not found")
	}

	// Check imports
	if len(result.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(result.Imports))
	}

	// Check call sites
	if len(result.CallSites) < 1 {
		t.Errorf("Expected at least 1 call site, got %d", len(result.CallSites))
	}
}

// TestPythonParser tests Python language parsing
func TestPythonParser(t *testing.T) {
	sourceCode := []byte(`
from typing import List
import json

class User:
	def __init__(self, name: str):
		self.name = name

	def greet(self):
		print(f"Hello, {self.name}")

def create_user(name: str) -> User:
	return User(name)
`)

	p, err := New(LangPython)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	result, err := p.Parse(context.Background(), sourceCode)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Check symbols
	if len(result.Symbols) < 3 {
		t.Errorf("Expected at least 3 symbols, got %d", len(result.Symbols))
	}

	// Check for User class
	foundClass := false
	for _, sym := range result.Symbols {
		if sym.Name == "User" && sym.Kind == "class" {
			foundClass = true
			break
		}
	}
	if !foundClass {
		t.Error("User class not found")
	}

	// Check imports
	if len(result.Imports) < 2 {
		t.Errorf("Expected at least 2 imports, got %d", len(result.Imports))
	}

	// Check call sites
	if len(result.CallSites) < 1 {
		t.Errorf("Expected at least 1 call site, got %d", len(result.CallSites))
	}
}

// TestDetectLanguage tests language detection
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want Language
	}{
		{"test.go", LangGo},
		{"test.ts", LangTypeScript},
		{"test.tsx", LangTypeScript},
		{"test.js", LangJavaScript},
		{"test.jsx", LangJavaScript},
		{"test.py", LangPython},
		{"test.txt", LangUnknown},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

package main

import (
    "fmt"
    "time"
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/launcher"
)

func main() {
    fmt.Println("Launching browser to check LinkedIn login page...")
    
    path, _ := launcher.LookPath()
    u := launcher.New().Bin(path).Headless(false).MustLaunch()
    browser := rod.New().ControlURL(u).MustConnect()
    defer browser.MustClose()
    
    page := browser.MustPage("https://www.linkedin.com/login")
    page.MustWaitLoad()
    time.Sleep(3 * time.Second)
    
    // Try to find input elements and print their attributes
    inputs, err := page.Elements("input")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    
    fmt.Printf("\nFound %d input elements:\n\n", len(inputs))
    for i, input := range inputs {
        id, _ := input.Attribute("id")
        name, _ := input.Attribute("name")
        inputType, _ := input.Attribute("type")
        autocomplete, _ := input.Attribute("autocomplete")
        placeholder, _ := input.Attribute("placeholder")
        
        fmt.Printf("Input %d:\n", i+1)
        if id != nil { fmt.Printf("  id: %s\n", *id) }
        if name != nil { fmt.Printf("  name: %s\n", *name) }
        if inputType != nil { fmt.Printf("  type: %s\n", *inputType) }
        if autocomplete != nil { fmt.Printf("  autocomplete: %s\n", *autocomplete) }
        if placeholder != nil { fmt.Printf("  placeholder: %s\n", *placeholder) }
        fmt.Println()
    }
    
    fmt.Println("Keeping browser open for 10 seconds...")
    time.Sleep(10 * time.Second)
}

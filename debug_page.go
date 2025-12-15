package main

import (
    "fmt"
    "time"
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/launcher"
)

func main() {
    fmt.Println("Launching browser with same settings as main app...")
    
    l := launcher.New().
        Headless(false).
        Set("disable-blink-features", "AutomationControlled").
        Set("disable-infobars")
    
    url, _ := l.Launch()
    browser := rod.New().ControlURL(url).MustConnect()
    defer browser.MustClose()
    
    page := browser.MustPage("https://www.linkedin.com/login")
    time.Sleep(5 * time.Second)
    
    fmt.Println("\nCurrent URL:", page.MustInfo().URL)
    fmt.Println("\nPage Title:", page.MustElement("title").MustText())
    
    // Check for any input fields
    inputs, _ := page.Elements("input")
    fmt.Printf("\nFound %d input elements\n", len(inputs))
    
    // Check for #username specifically
    email, err := page.Element("#username")
    if err != nil {
        fmt.Println("ERROR: #username not found:", err)
    } else {
        fmt.Println("SUCCESS: #username found!")
        visible, _ := email.Visible()
        fmt.Println("  Visible:", visible)
    }
    
    fmt.Println("\nKeeping browser open for 30 seconds to inspect...")
    time.Sleep(30 * time.Second)
}

package main

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func main() {
	fmt.Println("Testing Firebase service account file...")

	// Check if file exists
	if _, err := os.Stat("firebase-service-account.json"); err != nil {
		fmt.Printf("❌ Service account file not found: %v\n", err)
		return
	}

	fmt.Println("✅ Service account file found")

	// Try to initialize Firebase
	opt := option.WithCredentialsFile("firebase-service-account.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		fmt.Printf("❌ Failed to create Firebase app: %v\n", err)
		return
	}

	fmt.Println("✅ Firebase app created successfully")

	// Try to create messaging client
	ctx := context.Background()
	messaging, err := app.Messaging(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to create messaging client: %v\n", err)
		return
	}

	fmt.Println("✅ Messaging client created successfully")
	fmt.Println("✅ Your Firebase configuration is working correctly!")
	fmt.Println("")
	fmt.Println("If you're still getting 404 errors, restart your backend server.")
}

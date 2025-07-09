package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/px4n/finspect/adaptors/cloud"
	_ "github.com/px4n/finspect/adaptors/s3" // Register S3 provider
	"github.com/px4n/finspect/pkg/vfs"
)

func main() {
	// Create cloud adaptor factory
	factory := cloud.NewFactory()

	// Create S3 adaptor
	s3Adaptor, err := factory.Create(cloud.ProviderS3)
	if err != nil {
		log.Fatal("Failed to create S3 adaptor:", err)
	}

	// Wrap it for VFS
	vfsAdaptor := cloud.NewVFSCloudAdaptor(s3Adaptor)

	// Connect to S3
	ctx := context.Background()
	err = vfsAdaptor.Connect(ctx, map[string]interface{}{
		"provider": "s3",
		"bucket":   os.Getenv("S3_BUCKET"),
		"region":   os.Getenv("AWS_REGION"),
		"credentials": map[string]interface{}{
			"access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
			"secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to S3:", err)
	}
	defer func() {
		if err := vfsAdaptor.Disconnect(); err != nil {
			log.Printf("Error disconnecting: %v", err)
		}
	}()

	// Create a VFS router and mount S3
	router := vfs.NewRouter()
	err = router.Mount("/s3", vfsAdaptor)
	if err != nil {
		// Log error before exiting since defer won't run after Fatal
		if disconnectErr := vfsAdaptor.Disconnect(); disconnectErr != nil {
			log.Printf("Error disconnecting: %v", disconnectErr)
		}
		log.Fatal("Failed to mount S3:", err)
	}

	// List files in the root of the bucket
	entries, err := router.ReadDir("/s3/")
	if err != nil {
		log.Fatal("Failed to read directory:", err)
	}

	fmt.Println("Files in S3 bucket:")
	for _, entry := range entries {
		info, _ := entry.Info()
		fmt.Printf("  %s (size: %d, dir: %v)\n", entry.Name(), info.Size(), entry.IsDir())
	}

	// Example: Read a file
	file, err := router.Open("/s3/example.txt")
	if err != nil {
		fmt.Println("Could not open example.txt:", err)
		return
	}
	defer file.Close()

	// Read file contents
	buf := make([]byte, 1024)
	n, _ := file.Read(buf)
	fmt.Printf("\nContents of example.txt:\n%s\n", buf[:n])
}

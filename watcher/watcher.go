package main

// Import needed packages
import (
	"encoding/json"
	"github.com/boltdb/bolt"
	"gopkg.in/fsnotify.v1"
	"log"
	"time"
)

type WatchEvent struct {
	Type fsnotify.Event
	File string
}

func ExampleNewWatcher(db *bolt.DB) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {

					watchEvent := &WatchEvent{
						Type: event,
						File: event.Name,
					}
					db.Update(func(tx *bolt.Tx) error {
						b, err := tx.CreateBucketIfNotExists([]byte("events"))
						if err != nil {
							return err
						}
						encoded, err := json.Marshal(watchEvent)
						if err != nil {
							return err
						}
						return b.Put([]byte(time.Now().Format(time.RFC3339)), encoded)
					})
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func main() {
	db, err := bolt.Open("/tmp.db", 0644, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ExampleNewWatcher(db)
}

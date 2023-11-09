package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"github.com/gin-gonic/gin"
)

var (
	option1  string
	option2  string
	question string
	db       *badger.DB
	votedIPs = make(map[string]bool)
	mu       sync.Mutex
)

func initDB() {
	var err error
	opts := badger.DefaultOptions("badger-db")
	db, err = badger.Open(opts)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	// Initialize keys with initial values if they do not exist
	err = db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(option1))
		if err != nil && err == badger.ErrKeyNotFound {
			// Key not found, set initial value
			err = txn.Set([]byte(option1), []byte{0})
			if err != nil {
				return err
			}
		}

		_, err = txn.Get([]byte(option2))
		if err != nil && err == badger.ErrKeyNotFound {
			// Key not found, set initial value
			err = txn.Set([]byte(option2), []byte{0})
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal("Error initializing keys:", err)
	}
}

func main() {
	flag.StringVar(&option1, "option1", "Option 1", "Text for Option 1")
	flag.StringVar(&option2, "option2", "Option 2", "Text for Option 2")
	flag.StringVar(&question, "question", "Vote for Your Favorite", "Question for the voting poll")
	flag.Parse()

	initDB()
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		var option1Count, option2Count int
		err := db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(option1))
			if err == nil {
				err = item.Value(func(val []byte) error {
					option1Count = int(val[0])
					return nil
				})
			}
			item, err = txn.Get([]byte(option2))
			if err == nil {
				err = item.Value(func(val []byte) error {
					option2Count = int(val[0])
					return nil
				})
			}
			return err
		})
		if err != nil {
			option1Count, option2Count = 0, 0
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Question":     question,
			"Option1":      option1,
			"Option2":      option2,
			"Option1Votes": option1Count,
			"Option2Votes": option2Count,
		})
	})

	r.POST("/vote", func(c *gin.Context) {
		clientIP := c.ClientIP() // Get client's IP address
		mu.Lock()
		defer mu.Unlock()

		if votedIPs[clientIP] {
			// User has already voted
			c.JSON(http.StatusForbidden, gin.H{"error": "You have already voted."})
			return
		}

		option := c.PostForm("vote")

		// Validate the selected option
		if option != option1 && option != option2 {
			log.Println("Invalid option:", option)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option"})
			return
		}

		// Update the votes
		key := []byte(option)
		err := db.Update(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err == nil {
				var count int
				err = item.Value(func(val []byte) error {
					count = int(val[0])
					return nil
				})
				if err == nil {
					err = txn.Set(key, []byte{byte(count + 1)})
				}
			}
			return err
		})

		if err != nil {
			log.Println("Error updating votes:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		// Mark the user's IP as voted
		votedIPs[clientIP] = true

		// Retrieve the updated vote counts
		option1Count := dbGetVotes(option1)
		option2Count := dbGetVotes(option2)

		// Render the index.html template with the updated vote counts
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Question":     question,
			"Option1":      option1,
			"Option2":      option2,
			"Option1Votes": option1Count,
			"Option2Votes": option2Count,
		})
	})

	fmt.Println("Server is running on :8099")
	r.Run(":8099")
}

func dbGetVotes(option string) int {
	var count int
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(option))
		if err == nil {
			err = item.Value(func(val []byte) error {
				count = int(val[0])
				return nil
			})
		}
		return err
	})
	if err != nil {
		count = 0
	}
	return count
}

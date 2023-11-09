package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "votes.db")
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS votes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            option TEXT NOT NULL,
            count INTEGER NOT NULL
        );
    `)
	if err != nil {
		log.Fatal("Error creating table:", err)
	}
}

func main() {
	initDB()
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		var option1Count, option2Count int
		err := db.QueryRow("SELECT count FROM votes WHERE option = 'Option 1'").Scan(&option1Count)
		if err != nil {
			option1Count = 0
		}
		err = db.QueryRow("SELECT count FROM votes WHERE option = 'Option 2'").Scan(&option2Count)
		if err != nil {
			option2Count = 0
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Option1Votes": option1Count,
			"Option2Votes": option2Count,
		})
	})

	r.POST("/vote", func(c *gin.Context) {
		option := c.PostForm("vote")

		_, err := db.Exec("UPDATE votes SET count = count + 1 WHERE option = ?", option)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	fmt.Println("Server is running on :8099")
	r.Run(":8099")
}

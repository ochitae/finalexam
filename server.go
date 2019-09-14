package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type MyApp struct {
	DB *sql.DB
}

type Customer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

func authMiddleware(c *gin.Context) {
	fmt.Println("This is authMiddleware")
	token := c.GetHeader("Authorization")
	fmt.Println("token: ", token)
	if token != "token2019" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	c.Next()
	fmt.Println("After authMiddleware")
}

func (app MyApp) createTableCustomer() {
	createTb := `
	CREATE TABLE IF NOT EXISTS customer (
		id SERIAL PRIMARY KEY,
		name TEXT,
		email TEXT,
		status TEXT
	);
	`
	_, err := app.DB.Exec(createTb)
	if err != nil {
		log.Fatal("Cannot create table", err)
	}
}

func (app MyApp) createCustomerHandler(c *gin.Context) {
	var cs Customer
	err := c.ShouldBindJSON(&cs)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	row := app.DB.QueryRow(
		"insert into customer (name, email, status) values ($1, $2, $3) returning id",
		cs.Name, cs.Email, cs.Status,
	)
	err = row.Scan(&cs.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, cs)
}

func (app MyApp) getCustomerHandler(c *gin.Context) {
	id := c.Param("id")
	var cs Customer
	stmt, err := app.DB.Prepare("select id, name, email, status from customer where id=$1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	row := stmt.QueryRow(id)

	err = row.Scan(&cs.ID, &cs.Name, &cs.Email, &cs.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, cs)
}

func (app MyApp) getCustomersHandler(c *gin.Context) {
	var cs Customer
	var css = []Customer{}
	stmt, err := app.DB.Prepare("select id, name, email, status from customer")
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := stmt.Query()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	for rows.Next() {
		err = rows.Scan(&cs.ID, &cs.Name, &cs.Email, &cs.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		css = append(css, cs)

	}
	c.JSON(http.StatusOK, css)
}

func (app MyApp) updateCustomerHandler(c *gin.Context) {
	id := c.Param("id")
	var cs Customer
	err := c.ShouldBindJSON(&cs)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	stmt, err := app.DB.Prepare("update customer set name=$2, email=$3, status=$4 where id=$1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := stmt.Exec(id, cs.Name, cs.Email, cs.Status); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	app.getCustomerHandler(c)
}

func (app MyApp) deleteCustomerHandler(c *gin.Context) {
	id := c.Param("id")
	stmt, err := app.DB.Prepare("delete from customer where id=$1")
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := stmt.Exec(id); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "customer deleted"})
}

func main() {
	r := gin.Default()
	r.Use(authMiddleware)

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Cannot connect DB", err)
	}

	app := MyApp{db}
	defer app.DB.Close()

	app.createTableCustomer()

	r.POST("/customers", app.createCustomerHandler)
	r.GET("/customers/:id", app.getCustomerHandler)
	r.GET("/customers", app.getCustomersHandler)
	r.PUT("/customers/:id", app.updateCustomerHandler)
	r.DELETE("/customers/:id", app.deleteCustomerHandler)
	r.Run(":2019")
}

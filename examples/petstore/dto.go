package main

import "time"

type Pet struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Status    string   `json:"status" enum:"available,pending,sold"`
	Category  Category `json:"category"`
	Tags      []Tag    `json:"tags"`
	PhotoURLs []string `json:"photoUrls"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type FindPetsByStatusRequest struct {
	Status string `query:"status" enum:"available,pending,sold"`
}

type FindPetsByTagsRequest struct {
	Tags []string `query:"tags"`
}

type UpdatePetWithFormRequest struct {
	ID     int    `path:"petId" required:"true"`
	Name   string `formData:"name" required:"true"`
	Status string `formData:"status" enum:"available,pending,sold"`
}

type DeletePetRequest struct {
	ID     int    `path:"petId" required:"true"`
	ApiKey string `header:"api_key"`
}

type Order struct {
	ID       int       `json:"id"`
	PetID    int       `json:"petId"`
	Quantity int       `json:"quantity"`
	ShipDate time.Time `json:"shipDate"`
	Status   string    `json:"status" enum:"placed,approved,delivered"`
	Complete bool      `json:"complete"`
}

type User struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Phone      string `json:"phone"`
	UserStatus int    `json:"userStatus" enum:"0,1,2"`
}

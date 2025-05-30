package schemas

import "time"

type ID string
type Name string
type Address string
type PictureUrl string

type TasksSearchParams struct {
	Status string
	Limit  uint16
	Page   uint16
}

type TaskUpdate struct {
	AssigneeId string `bson:"assigneeId" json:"assigneeId"`
}

type Tasks struct {
	Pagination Pagination `json:"pagination"`
	Tasks      []Task     `json:"tasks"`
}

type Task struct {
	ID           ID           `bson:"id" json:"id"`
	Name         Name         `bson:"name" json:"name"`
	Ops          Ops          `bson:"ops" json:"ops"`
	Organisation Organisation `bson:"organisation" json:"organisation"`
	Shifts       []Shift      `bson:"shifts" json:"shifts"`
}

type Ops struct {
	Firstname string `bson:"firstname" json:"firstname"`
	Lastname  string `bson:"lastname" json:"lastname"`
}

type Organisation struct {
	Name       Name   `bson:"name" json:"name"`
	Address    string `bson:"address" json:"address"`
	PictureUrl string `bson:"pictureUrl" json:"pictureUrl"`
}

type Shift struct {
	ID         ID         `bson:"id" json:"id"`
	StartDate  *time.Time `bson:"startDate" json:"startDate"`
	EndDate    *time.Time `bson:"endDate" json:"endDate"`
	Slots      Slots      `bson:"slots" json:"slots"`
	Applicants uint16     `bson:"applicants" json:"applicants"`
}

type Slots struct {
	Filled uint8 `bson:"filled" json:"filled"`
	Total  uint8 `bson:"total" json:"total"`
}

type Pagination struct {
	Limit uint16 `json:"limit"`
	Page  uint16 `json:"page"`
}

type BodyError struct {
	Error string `json:"error"`
}

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/JeromeM/TestTechGoMongo/schemas"
	"github.com/kataras/golog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client *mongo.Client
	Tasks  *mongo.Collection
}

const (
	DEFAULT_LIMIT = 20
	DEFAULT_SKIP  = 0
	MAX_LIMIT     = 100
)

// Open a connection to MongoDB
func NewMongoClient(uri string, dbName string) *MongoClient {
	const (
		TASK_COLLECTION = "tasks"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		golog.Fatalf("could not instantiate mongo client: %v", err)
	}

	tasks := client.Database(dbName).Collection(TASK_COLLECTION)
	golog.Infof("Connected to %v collection on Mongo!", TASK_COLLECTION)

	return &MongoClient{
		Client: client,
		Tasks:  tasks,
	}
}

// Get all tasks list (with filter)
func (m *MongoClient) GetTasks(params *schemas.TasksSearchParams) ([]schemas.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mainPipeline := GetPipeline()
	if filter := Filter(params); filter != nil {
		mainPipeline = append(mainPipeline, filter)
	}
	skip, limit := Paginate(params)
	mainPipeline = append(mainPipeline, skip, limit)

	cur, err := m.Tasks.Aggregate(ctx, mainPipeline)
	if err != nil {
		return nil, fmt.Errorf("could not find tasks from Mongo: %v", err)
	}
	defer cur.Close(ctx)

	golog.Infof("Fetching tasks from Mongo")

	res := make([]schemas.Task, 0)

	for cur.Next(ctx) {
		var e schemas.Task
		err = cur.Decode(&e)
		if err != nil {
			return nil, fmt.Errorf("could not decode Mongo response: %v", err)
		}

		res = append(res, e)
	}

	if err = cur.Err(); err != nil {
		return nil, fmt.Errorf("error while getting sources: %v", err)
	}

	return res, nil
}

func GetPipeline() mongo.Pipeline {
	// Pipeline might have been more optimized .. There are three lookups ..
	// Fortunately the datasets are not so huge .. I've worked on collections of more than 50M documents
	// and lookups were still working well.
	pipeline := mongo.Pipeline{
		// 1 : Add project to limit document size
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "id", Value: "$_id"},
				{Key: "name", Value: "$alias"},
				{Key: "organisationId", Value: 1},
				{Key: "shiftIds", Value: 1},
				{Key: "assigneeId", Value: 1},
			}},
		},
		// 2 : Lookup organisation informations
		bson.D{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "orgas"},
				{Key: "localField", Value: "organisationId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "pipeline", Value: bson.A{
					bson.D{
						{Key: "$project", Value: bson.D{
							{Key: "_id", Value: 0},
							{Key: "name", Value: 1},
							{Key: "address", Value: 1},
							{Key: "pictureUrl", Value: "$logoUrl"},
						}},
					},
				}},
				{Key: "as", Value: "organisation"},
			}},
		},
		// 3 : Lookup users informations
		bson.D{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "assigneeId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "pipeline", Value: bson.A{
					bson.D{
						{Key: "$project", Value: bson.D{
							{Key: "_id", Value: 0},
							{Key: "firstname", Value: "$profile.FirstName"},
							{Key: "lastname", Value: "$profile.LastName"},
						}},
					},
				}},
				{Key: "as", Value: "ops"},
			}},
		},
		// 4 : Lookup shifts informations
		//     And add some size calculations
		bson.D{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "shifts"},
				{Key: "localField", Value: "shiftIds"},
				{Key: "foreignField", Value: "_id"},
				{Key: "pipeline", Value: bson.A{
					bson.D{
						{Key: "$match", Value: bson.D{
							{Key: "status", Value: bson.D{
								{Key: "$ne", Value: "cancelled"},
							}},
						}},
					},
					bson.D{
						{Key: "$addFields", Value: bson.D{
							{Key: "applicants", Value: bson.D{
								{Key: "$size", Value: "$availableSiderIds"},
							}},
							{Key: "slots.filled", Value: bson.D{
								{Key: "$size", Value: "$hiredSiderIds"},
							}},
							{Key: "slots.total", Value: "$slots"},
						}},
					},
					bson.D{
						{Key: "$project", Value: bson.D{
							{Key: "_id", Value: 0},
							{Key: "id", Value: "$_id"},
							{Key: "startDate", Value: bson.D{
								{Key: "$toDate", Value: "$time.startDate"},
							}},
							{Key: "endDAte", Value: bson.D{
								{Key: "$toDate", Value: "$time.endDate"},
							}},
							{Key: "slots", Value: 1},
							{Key: "applicants", Value: 1},
						}},
					},
				}},
				{Key: "as", Value: "shifts"},
			}},
		},
		// 5 : Just check if there is at least 1 shift
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "shifts.0", Value: bson.D{
					{Key: "$exists", Value: true},
				}},
			}},
		},
		// 6 : Remove the lookup arrays
		bson.D{
			{Key: "$addFields", Value: bson.D{
				{Key: "organisation", Value: bson.D{
					{Key: "$arrayElemAt", Value: bson.A{
						"$organisation",
						0,
					}},
				}},
				{Key: "ops", Value: bson.D{
					{Key: "$arrayElemAt", Value: bson.A{
						"$ops",
						0,
					}},
				}},
			}},
		},
		// 7 : Remove useless fields used for lookups
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "organisationId", Value: 0},
				{Key: "shiftIds", Value: 0},
				{Key: "assigneeId", Value: 0},
			}},
		},
	}

	return pipeline
}

/*
Add a step to the aggregate to filter tasks with shifts dates
- Upcoming: All the shifts are in the future.
- Ongoing: Some shifts are in the past, some in the future. Return all shifts (default)
- Done: All the shifts are in the past.
*/
func Filter(params *schemas.TasksSearchParams) bson.D {
	var dateParam bson.D

	// I haven't added ongoing (except if I have misunderstood something), as it seems
	// it's all the shifts availables
	switch params.Status {
	case "upcoming":
		dateParam = bson.D{
			{Key: "$gte", Value: bson.A{
				"$$shift.startDate", time.Now(),
			}},
		}
	case "done":
		dateParam = bson.D{
			{Key: "$lte", Value: bson.A{
				"$$shift.startEnd", time.Now(),
			}},
		}
	default:
		return nil
	}
	pipeline := bson.D{
		// Keep only documents where all elements from the shifts array
		// matches the requested status
		{Key: "$match", Value: bson.D{
			{Key: "$expr", Value: bson.D{
				{Key: "$allElementsTrue", Value: bson.D{
					{Key: "$map", Value: bson.D{
						{Key: "input", Value: "$shifts"},
						{Key: "as", Value: "shift"},
						{Key: "in", Value: bson.D{
							{Key: "$and", Value: bson.A{
								dateParam,
							}},
						}},
					}},
				}},
			}},
		}},
	}

	return pipeline
}

func Paginate(params *schemas.TasksSearchParams) (bson.D, bson.D) {
	skip, limit := validatePagination(params)
	skipPipeline := bson.D{
		{Key: "$skip", Value: skip},
	}

	limitPipeline := bson.D{
		{Key: "$limit", Value: limit},
	}

	return skipPipeline, limitPipeline
}

func GetPagination(params *schemas.TasksSearchParams) schemas.Pagination {
	_, limit := validatePagination(params)
	return schemas.Pagination{
		Limit: limit,
		Page:  params.Page,
	}

}

func validatePagination(params *schemas.TasksSearchParams) (uint16, uint16) {
	var skip uint16 = DEFAULT_SKIP
	var limit uint16 = DEFAULT_LIMIT

	if params.Limit > 0 {
		limit = params.Limit
		if limit > MAX_LIMIT {
			golog.Warn(fmt.Sprintf("Used %d (max limit) instead of %d", MAX_LIMIT, limit))
			limit = MAX_LIMIT
		}
	}
	if params.Page > 1 {
		skip = limit * params.Page
	}

	return skip, limit
}

func (m *MongoClient) UpdateOne(taskID string, params schemas.TaskUpdate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	filter := bson.M{"_id": taskID}

	now := time.Now()
	// Format the date as it's a string in the database due to the import
	updated_at := now.UTC().Format("2006-01-01T15:04:05.000Z")
	doc := bson.D{{Key: "$set", Value: bson.D{
		{Key: "assigneeId", Value: params.AssigneeId},
		{Key: "updatedAt", Value: updated_at},
	}}}

	_, err := m.Tasks.UpdateOne(ctx, filter, doc)

	if err != nil {
		return fmt.Errorf("error while updating task %v assignee in mongo: %v", taskID, err)
	}

	golog.Infof("Successfully updated task %v in mongo", taskID)

	return nil
}

# SELECTOR

Selector is used for fetching tasks and shifts for showing in the front-end.

## INSTALLING AND RUNNING

An environment file (`.env`) must be added at the root of the project. A sample is available and a fullfiled file is in the *kustomize* folder.  
You can build with `make build` or just simply run with `make run`.

## USAGE

> **GET** /tasks

Fetch all the tasks  

*URL Parameters*  
- `limit` : *int* | Number of tasks to return (default 20, max 100)
- `page` : *int* | Page to go to (default 1)
- `status` : *string* | Only show tasks according to a date range :  
   - `upcoming` : All the shifts are in the future
   - `ongoing` : Some shifts are in the past, some in the future
   - `done` : All the shifts are in the past

*Return*
```
{
    "pagination": {
        "limit": 1,
        "page": 1
    },
    "tasks": [
        {
            "id": "ta_2HwbQsShYubAbDpQ4pODWseJhzK",
            "name": "Vendeur en prêt-à-porter (F/H) - Sheriff",
            "ops": {
                "firstname": "Alfred",
                "lastname": "Jacqueline"
            },
            "organisation": {
                "name": "WaterSmart Software",
                "address": "1796  rue du Cheminot-Coquelin , Le Mans, Pyrénées-Atlantiques (64) 16558",
                "pictureUrl": "https://picsum.photos/400/400"
            },
            "shifts": [
                {
                    "id": "sh_2HwbXTimfrUJ5SknbRmhdJgtmF0",
                    "startDate": "2023-01-14T10:00:00Z",
                    "endDate": null,
                    "slots": {
                        "filled": 1,
                        "total": 1
                    },
                    "applicants": 1
                }
            ]
        }
    ]
}
```

***
> **PATCH** /tasks/{TASK_ID}

Update a task

*Parameters*
- `TASK_ID` : ID of the task to patch

*JSON Payload*

Update the current assigneeID by the one in the payload
```
{
    "assigneeId": "XXXXX"
}
```

*Return*

```
{
    "assigneeId": "057cdd28be42f2096357a6f9e93bb234362672744bde8df3816c37c4a90e0219"
}
```

## MISC  

> **Kubernetes**

There is a .kustomize folder to run in Kubernetes.  
A `.env` file is there, but the password is hidden. I'll give it to you at the same time I will give you access to this repository.

> **Database selection**

I'm using a MongoDB database, as it can perform very well on big datasets.

First thing I've done is to create a database on an Atlas cluster and import the given JSON files. 
This way I easily have five collections with all the informations needed for the tool.
Problem is : I can't set date to datetime format in Mongo when importing JSON (via Studio 3T tool).

Indices have been added to shifts collection on time.startDate and time.endDate as we will use these fields in the request.

I have created an access to my Atlas cluster ([more here](#mongodocker)) for storing the database and collections.

> **Tasks aggregate**

I'm going for an optimised aggregate with MongoDB to fetch the tasks and their relative informations (organisation and shifts).

```
[
    {
        $project: {
            "_id": 0,
            "id": "$_id",
            "name": "$alias",
            "organisationId": 1,
            "shiftIds": 1,
            "assigneeId": 1
        }
    },
    {
        $lookup: {
            from: "orgas",
            localField: "organisationId",
            foreignField: "_id",
            pipeline: [
                {
                    $project: {
                        "_id": 0,
                        "name": 1,
                        "address": 1,
                        "pictureUrl": "$logoUrl"
                    }
                }
            ],
            as: "organisation"
        }
    },
    {
        $lookup: {
            from: "users",
            localField: "assigneeId",
            foreignField: "_id",
            pipeline: [
                {
                    $project: {
                        "_id": 0,
                        "firstname": "$profile.FirstName",
                        "lastname": "$profile.LastName"
                    }
                }
            ],
            as: "ops"
        }
    },
    {
        $lookup: {
            from: "shifts",
            localField: "shiftIds",
            foreignField: "_id",
            pipeline: [
                {
                    $match: {
                        "status" : { $ne : "cancelled" }
                    }
                },
                {
                    $addFields: {
                        "applicants": { $size: "$availableSiderIds" },
                        "slots.filled": { $size: "$hiredSiderIds" },
                        "slots.total": "$slots",
                        "time.startDate": { $toDate: "$time.startDate" },
                        "time.endDate": { $toDate: "$time.endDate" }
                    }
                },
                        {
                    $project: {
                        "_id": 0,
                        "id": "$_id",
                        "startDate": "$time.startDate",
                        "endDate": "$time.endDate",
                        "slots": 1,
                        "applicants": 1
                    }
                },
            ],
            as: "shifts"
        }
    },
    {
        $match: {
            "shifts.0": {
                $exists: true
            }
        }
    },
    {
        $addFields: {
            "organisation": {
                $arrayElemAt: ["$organisation", 0]
            },
            "ops": {
                $arrayElemAt: ["$ops", 0]
            }
        }
    },
    {
        $project: {
            "organisationId": 0,
            "shiftIds": 0,
            "assigneeId": 0
        }
    },
    {
        $match: {
          $expr: {
            $allElementsTrue: {
              $map: {
                input: "$shifts.startDate",
                as: "date",
                in: {
                  $and: [
                    { $gte: ["$$date", new Date("2023-01-19")] }
                  ]
                }
              }
            }
          }
        }
    }
]
```

## MORE THOUGHTS AND INTENTIONS

### Pagination

I don't have added the "total" in the pagination, as it require to run the aggregate one more time to have an accurate result, and I'm afraid it will slow the request.

At first I was getting all the results and paginate a list, but the full request was taking between 1.5s and 1.7s, it was too slow.  
With the **$skip** and **$limit** parameters we're down to 150ms for each request.  

I guess the front will never show more than 100 tasks (although I think it's too much).

### Docker

<a name=mongodocker></a>I was ready to Dockerize MongoDB database, but I was unsure how to automate the import of the JSON files and create the new indices.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	cli "github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var collection *mongo.Collection
var ctx = context.TODO()

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	check(err)

	err = client.Ping(ctx, nil)
	check(err)

	collection = client.Database("tasker").Collection("tasks")
}

func main() {
	app := &cli.App{
		Name:  "tasker",
		Usage: "A simple CLI program to manage your tasks",
		Action: func(c *cli.Context) error {

			tasks, err := getPending()
			if err != nil {
				if err == mongo.ErrNoDocuments {
					fmt.Println("Nothing to see here.\n Run `add <task>` to add a new task")
					return nil
				}
				return err
			}

			printTasks(tasks)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "add a task to the list",
				Action: func(c *cli.Context) error {
					str := c.Args().First()

					if str == "" {
						return errors.New("Cannot add a empty task")
					}

					task := Task{
						ID:        primitive.NewObjectID(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Text:      str,
						Completed: false,
					}

					return createTask(&task)

				},
			},
			{
				Name:    "all",
				Aliases: []string{"l"},
				Usage:   "list all tasks",
				Action: func(c *cli.Context) error {
					tasks, err := getAll()
					if err != nil {
						if err == mongo.ErrNoDocuments {
							fmt.Print("Nothing to see here. \n Run `add 'task' to add a new task")
							return nil
						}
						return err
					}

					printTasks(tasks)
					return nil
				},
			},
			{
				Name:    "done",
				Aliases: []string{"d"},
				Usage:   "complete a task on the list",
				Action: func(c *cli.Context) error {
					text := c.Args().First()
					return completeTask(text)
				},
			},
			{
				Name:    "finished",
				Aliases: []string{"f"},
				Usage:   "shows all the completed tasks",
				Action: func(c *cli.Context) error {

					tasks, err := getFinished()
					if err != nil {
						if err == mongo.ErrNoDocuments {
							fmt.Println("Nothing to see here.\n Run `add <task>` to add a new task")
							return nil
						}
						return err
					}

					printTasks(tasks)
					return nil
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm"},
				Usage:   "deletes the task on the list",
				Action: func(c *cli.Context) error {
					text := c.Args().First()
					return deleteTask(text)
				},
			},
		},
	}

	err := app.Run(os.Args)
	check(err)
}

//Task is a struct that represents a single Task in the database.
type Task struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"created_at,omitempty"`
	UpdatedAt time.Time          `bson:"updated_at,omitempty"`
	Text      string             `bson:"text,omitempty"`
	Completed bool               `bson:"completed"`
}

func createTask(task *Task) error {

	_, err := collection.InsertOne(ctx, task)

	return err
}

func getAll() ([]*Task, error) {
	filter := bson.D{{}}
	return filterTasks(filter)
}

func filterTasks(filter interface{}) ([]*Task, error) {
	var tasks []*Task

	cur, err := collection.Find(ctx, filter)
	check(err)

	for cur.Next(ctx) {
		var t Task
		err := cur.Decode(&t)
		check(err)

		tasks = append(tasks, &t)
	}

	if err := cur.Err(); err != nil {
		return tasks, err
	}

	cur.Close(ctx)

	if len(tasks) == 0 {
		return tasks, mongo.ErrNoDocuments
	}

	return tasks, nil
}

func completeTask(text string) error {
	filter := bson.D{primitive.E{Key: "text", Value: text}}

	update := bson.D{primitive.E{Key: "$set", Value: bson.D{primitive.E{Key: "completed", Value: true}}}}

	t := &Task{}

	return collection.FindOneAndUpdate(ctx, filter, update).Decode(t)
}

func getPending() ([]*Task, error) {

	filter := bson.D{primitive.E{Key: "completed", Value: false}}
	//  bson.D{primitive.E{Key: "$ne", Value: true}}}}
	return filterTasks(filter)
}

func getFinished() ([]*Task, error) {

	filter := bson.D{primitive.E{Key: "completed", Value: true}}
	//  bson.D{primitive.E{Key: "$ne", Value: true}}}}
	return filterTasks(filter)
}

func deleteTask(text string) error {

	filter := bson.D{primitive.E{Key: "text", Value: text}}

	res, err := collection.DeleteOne(ctx, filter)
	check(err)

	if res.DeletedCount == 0 {
		return errors.New("No Tasks were deleted")
	}
	return nil
}

func printTasks(tasks []*Task) {
	for i, v := range tasks {
		if v.Completed {
			color.Green.Printf("%d : %s\n", i+1, v.Text)
		} else {
			color.Yellow.Printf("%d : %s\n", i+1, v.Text)
		}
	}
}

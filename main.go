package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NebulousLabs/go-skynet/v2"
	"github.com/bluemediaapp/models"
	"github.com/bwmarrin/snowflake"
	"github.com/dhowden/tag"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	app    = fiber.New()
	client *mongo.Client
	config *Config

	mctx = context.Background()

	videosCollection        *mongo.Collection
	likedVideosCollection   *mongo.Collection
	usersCollection         *mongo.Collection
	watchedVideosCollection *mongo.Collection
)

type VideoUpload struct {
	Description string `form:"description"`
	Series      string `form:"series"`
}

func main() {
	config = &Config{
		port:     os.Getenv("port"),
		mongoUri: os.Getenv("mongo_uri"),
	}
	skyClient := skynet.New()

	snowflake.Epoch = time.Date(2020, time.January, 0, 0, 0, 0, 0, time.UTC).Unix()
	snowNode, _ := snowflake.NewNode(1)

	app.Get("/upload/:user_id", func(ctx *fiber.Ctx) error {
		userId, err := strconv.ParseInt(ctx.Params("user_id"), 10, 64)
		if err != nil {
			return err
		}
		uploadedVideo := new(VideoUpload)
		err = ctx.BodyParser(uploadedVideo)
		if err != nil {
			return err
		}
		if len(uploadedVideo.Description) > 255 {
			return errors.New("description is too long (max 255 characters)")
		}
		tags := make([]string, 0)
		splittedDescription := strings.Split(uploadedVideo.Description, " ")
		for _, keyword := range splittedDescription {
			if !strings.HasPrefix(keyword, "#") {
				continue
			}
			tag := strings.Replace(keyword, "#", "", 1)
			tags = append(tags, tag)
		}

		upload := make(map[string]io.Reader)
		file, err := ctx.FormFile("video_upload")
		if err != nil {
			return err
		}
		file_reader, err := file.Open()
		if err != nil {
			return err
		}

		video_meta, err := tag.ReadFrom(file_reader)
		if err != nil {
			return err
		}

		video_format := video_meta.Format()
		if video_format != "MP4" {
			return ctx.Status(400).SendString("We only accept mp4's")
		}

		upload["upload"] = file_reader
		skylink, err := skyClient.Upload(upload, skynet.DefaultUploadOptions)
		if err != nil {
			return err
		}

		video := models.DatabaseVideo{
			Id:          snowNode.Generate().Int64(),
			CreatorId:   userId,
			Description: uploadedVideo.Description,
			Series:      uploadedVideo.Series,
			Public:      true,
			Likes:       0,
			Tags:        tags,
			Modifiers:   make([]string, 0),
			StorageKey:  strings.Replace(skylink, "sia://", "", -1),
		}

		err = uploadVideo(video)
		if err != nil {
			return err
		}
		return ctx.JSON(video)

	})
	app.Get("/delete/:video_id/:user_id", func(ctx *fiber.Ctx) error {
		videoId, err := strconv.ParseInt(ctx.Params("video_id"), 10, 64)
		if err != nil {
			return err
		}

		video, err := getVideo(videoId)
		if err != nil {
			return ctx.Status(404).SendString("Video was not found.")
		}
		deleterId, err := strconv.ParseInt(ctx.Params("user_id"), 10, 64)
		if err != nil {
			return err
		}

		if deleterId != video.CreatorId {
			return ctx.Status(403).SendString("You did not create this video.")
		}

		videosCollection.DeleteOne(mctx, bson.D{{"_id", video.Id}})
		likedVideosCollection.DeleteMany(mctx, bson.D{{"video_id", video.Id}})
		watchedVideosCollection.DeleteMany(mctx, bson.D{{"_id", video.Id}})

		return ctx.SendString("Video deleted.")
	})

	initDb()
	log.Fatal(app.Listen(config.port))
}

func initDb() {
	// Connect mongo
	var err error
	client, err = mongo.NewClient(options.Client().ApplyURI(config.mongoUri))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Connect(mctx)
	if err != nil {
		log.Fatal(err)
	}

	// Setup tables
	db := client.Database("blue")
	videosCollection = db.Collection("video_metadata")
	likedVideosCollection = db.Collection("liked_videos")
	watchedVideosCollection = db.Collection("watched_videos")
}

func getVideo(videoId int64) (models.DatabaseVideo, error) {
	query := bson.D{{"_id", videoId}}
	rawVideo := videosCollection.FindOne(mctx, query)
	var video models.DatabaseVideo
	err := rawVideo.Decode(&video)
	if err != nil {
		return models.DatabaseVideo{}, err
	}
	return video, nil
}
func uploadVideo(video models.DatabaseVideo) error {
	_, err := videosCollection.InsertOne(mctx, video)
	if err != nil {
		return err
	}
	return nil
}

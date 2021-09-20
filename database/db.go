package database

import (
	"context"
	"regexp"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type DB struct {
	db     *mongo.Database
	client *mongo.Client
}

func NewDB(name string, uri string) (*DB, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Connect(context.TODO()); err != nil {
		return nil, err
	}
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, err
	}

	db := DB{
		db:     client.Database(name),
		client: client,
	}

	return &db, nil
}

func (db *DB) CreateCollection(name string) *mongo.Collection {
	c := db.db.Collection(name)
	if c != nil {
		return c
	}

	db.db.CreateCollection(context.TODO(), name)
	return db.db.Collection(name)
}

func (db *DB) NewsWithCodeExists(c *mongo.Collection, code string) bool {
	var res bson.M
	err := c.FindOne(context.TODO(), bson.M{"code": code}).Decode(&res)

	return err == nil
}

func (db *DB) Save(c *mongo.Collection, data interface{}) error {
	_, err := c.InsertOne(context.TODO(), data)
	if err != nil {
		logrus.Errorln(err)
	}
	return err
}

func (db *DB) GetNewsPageRegex(agency string) (*regexp.Regexp, error) {
	res := db.db.Collection("NewsWebsites").FindOne(context.TODO(), bson.M{"name": agency})
	if res.Err() != nil {
		return nil, res.Err()
	}

	data := struct {
		NewsPageRegex string `bson:"newsPageRegex"`
	}{}

	err := res.Decode(&data)
	if err != nil {
		return nil, err
	}
	return regexp.MustCompile(data.NewsPageRegex), nil
}

func (db *DB) GetNewsCodeRegex(agency string) (*regexp.Regexp, error) {
	res := db.db.Collection("NewsWebsites").FindOne(context.TODO(), bson.M{"name": agency})
	if res.Err() != nil {
		return nil, res.Err()
	}

	data := struct {
		NewsCodeRegex string `bson:"newsCodeRegex"`
	}{}
	err := res.Decode(&data)
	if err != nil {
		return nil, err
	}
	return regexp.MustCompile(data.NewsCodeRegex), nil
}
func (db *DB) GetArchiveURL(agency string) (string, error) {
	res := db.db.Collection("NewsWebsites").FindOne(context.TODO(), bson.M{"name": agency})
	if res.Err() != nil {
		return "", res.Err()
	}

	data := struct {
		ArchiveURL string `bson:"archiveURL"`
	}{}
	err := res.Decode(&data)
	if err != nil {
		return "", err
	}
	return data.ArchiveURL, nil
}

func (db *DB) InitDB() {
	db.db.CreateCollection(context.TODO(), "NewsWebsites")
	websites := db.db.Collection("NewsWebsites")

	websites.InsertMany(context.TODO(), []interface{}{
		bson.D{
			{"name", "Farsnews"},
			{"archiveURL", `https://www.farsnews.ir/archive?cat=-1&subcat=-1&date=%s&p=%d`},
			{"newsPageRegex", `https://www\.farsnews\.ir/.*news/\d+/.*`},
			{"newsCodeRegex", `\d+`},
		},
		bson.D{
			{"name", "ISNA"},
			{"archiveURL", `https://www.isna.ir/page/archive.xhtml?mn=%d&wide=0&dy=%d&ms=0&pi=%d&yr=%d`},
			{"newsPageRegex", `https://www\.isna\.ir/news/\d+/.*`},
			{"newsCodeRegex", `\d+`},
		},
		bson.D{
			{"name", "Tabnak"},
			{"archiveURL", `https://www.tabnak.ir/fa/archive?service_id=-1&sec_id=-1&cat_id=-1&rpp=10&from_date=1384/01/01&to_date=%s&p=%d`},
			{"newsPageRegex", `http(|s)://(www|ostanha)\.tabnak\w*\.ir/fa/news/\d+/.*`},
			{"newsCodeRegex", `\d+`},
		},
		bson.D{
			{"name", "Tasnim"},
			{"archiveURL", `https://www.tasnimnews.com/fa/archive?date=%s&sub=-1&service=-1&page=%d`},
			{"newsPageRegex", `https://www\.tasnimnews\.com/fa/news/\d{4}/\d{1,2}/\d{1,2}/\d{5,}/.*`},
			{"newsCodeRegex", `\d{5,}`},
		},
		bson.D{
			{"name", "YJC"},
			{"archiveURL", `https://www.yjc.news/fa/archive?service_id=-1&sec_id=-1&cat_id=-1&rpp=10&from_date=1390/01/01&to_date=%s&p=%d`},
			{"newsPageRegex", `https://www\.yjc\.news/fa/news/\d+/.*`},
			{"newsCodeRegex", `\d+`},
		},
	})

	db.db.CreateCollection(context.TODO(), "News")
	news := db.db.Collection("News")

	newsID := "news_id"
	unique := true
	news.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys: bson.D{{"code", 1}},
		Options: &options.IndexOptions{
			Name:   &newsID,
			Unique: &unique,
		},
	})
}

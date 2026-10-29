package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	auctionDurationEnv = "AUCTION_DURATION"
	auctionIntervalEnv = "AUCTION_INTERVAL"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	auctionEndTime := auctionEntity.Timestamp.Add(getAuctionDuration())
	go ar.closeAuctionWhenExpired(auctionEntity.Id, auctionEndTime)

	return nil
}

func (ar *AuctionRepository) closeAuctionWhenExpired(auctionId string, auctionEndTime time.Time) {
	timer := time.NewTimer(time.Until(auctionEndTime))
	defer timer.Stop()
	<-timer.C

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	filter := bson.M{
		"_id":    auctionId,
		"status": auction_entity.Active,
	}
	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	if _, err := ar.Collection.UpdateOne(ctx, filter, update); err != nil {
		logger.Error(fmt.Sprintf("Error trying to close auction with id = %s", auctionId), err)
	}
}

func getAuctionDuration() time.Duration {
	auctionDuration := os.Getenv(auctionDurationEnv)
	if auctionDuration == "" {
		auctionDuration = os.Getenv(auctionIntervalEnv)
	}

	duration, err := time.ParseDuration(auctionDuration)
	if err != nil {
		return time.Minute * 5
	}

	return duration
}

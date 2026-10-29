package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCreateAuctionAutomaticallyClosesAuctionAfterConfiguredDuration(t *testing.T) {
	t.Setenv(auctionDurationEnv, "200ms")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(getTestMongoURL()))
	if err != nil {
		t.Fatalf("failed to connect to mongodb: %v", err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("auction_auto_close_test")
	if err := database.Drop(ctx); err != nil {
		t.Fatalf("failed to clean test database: %v", err)
	}

	repository := NewAuctionRepository(database)
	auctionEntity, internalErr := auction_entity.CreateAuction(
		"Notebook",
		"Eletronicos",
		"Notebook usado em bom estado",
		auction_entity.Used,
	)
	if internalErr != nil {
		t.Fatalf("failed to create auction entity: %v", internalErr)
	}

	if internalErr := repository.CreateAuction(ctx, auctionEntity); internalErr != nil {
		t.Fatalf("failed to persist auction: %v", internalErr)
	}

	createdAuction, internalErr := repository.FindAuctionById(ctx, auctionEntity.Id)
	if internalErr != nil {
		t.Fatalf("failed to find created auction: %v", internalErr)
	}
	if createdAuction.Status != auction_entity.Active {
		t.Fatalf("expected auction to be active after creation, got %v", createdAuction.Status)
	}

	assertAuctionEventuallyCompleted(t, repository, auctionEntity.Id)
}

func assertAuctionEventuallyCompleted(t *testing.T, repository *AuctionRepository, auctionID string) {
	t.Helper()

	deadline := time.Now().Add(time.Second * 3)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		auctionEntity, internalErr := repository.FindAuctionById(ctx, auctionID)
		cancel()
		if internalErr != nil {
			t.Fatalf("failed to find auction while waiting for close: %v", internalErr)
		}

		if auctionEntity.Status == auction_entity.Completed {
			return
		}

		time.Sleep(time.Millisecond * 50)
	}

	t.Fatalf("expected auction %s to be completed after configured duration", auctionID)
}

func getTestMongoURL() string {
	if url := os.Getenv("AUCTION_TEST_MONGODB_URL"); url != "" {
		return url
	}

	return "mongodb://admin:admin@localhost:27017/auctions?authSource=admin"
}

package sql_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

// TestDefaultMySQLSchema checks if the SQL schema defined in DefaultMySQLSchema is correctly executed
// and if message marshaling works as intended.
func TestDefaultMySQLSchema(t *testing.T) {
	db := newMySQL(t)

	publisher, err := sql.NewPublisher(db, sql.PublisherConfig{
		SchemaAdapter:        sql.DefaultMySQLSchema{},
		AutoInitializeSchema: true,
	}, logger)
	require.NoError(t, err)

	subscriber, err := sql.NewSubscriber(db, sql.SubscriberConfig{
		SchemaAdapter:    sql.DefaultMySQLSchema{},
		OffsetsAdapter:   sql.DefaultMySQLOffsetsAdapter{},
		InitializeSchema: true,
	}, logger)
	require.NoError(t, err)

	testOneMessage(t, publisher, subscriber)
}

func TestDefaultMySQLSchema_implicit_commit_warning(t *testing.T) {
	db := newMySQL(t)
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	schemaAdapter := sql.DefaultMySQLSchema{}
	_, err = sql.NewPublisher(tx, sql.PublisherConfig{
		SchemaAdapter:        schemaAdapter,
		AutoInitializeSchema: true,
	}, logger)
	require.Error(t, err, "used auto schema initializing without a separate db handle for the adapter, "+
		"expected error from publisher constructor")
}

func TestDefaultMySQLSchema_implicit_commit(t *testing.T) {
	db := newMySQL(t)
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	schemaAdapter := sql.DefaultMySQLSchema{}
	_, err = sql.NewPublisher(tx, sql.PublisherConfig{
		SchemaAdapter:        schemaAdapter,
		AutoInitializeSchema: true,
	}, logger)
	require.Error(t, err, "expecting error with AutoInitializeSchema and db handle which is a tx")
}

// TestDefaultPostgreSQLSchema checks if the SQL schema defined in DefaultPostgreSQLSchema is correctly executed
// and if message marshaling works as intended.
func TestDefaultPostgreSQLSchema(t *testing.T) {
	db := newPostgreSQL(t)

	publisher, err := sql.NewPublisher(db, sql.PublisherConfig{
		SchemaAdapter:        sql.DefaultPostgreSQLSchema{},
		AutoInitializeSchema: true,
	}, logger)
	require.NoError(t, err)

	subscriber, err := sql.NewSubscriber(db, sql.SubscriberConfig{
		SchemaAdapter:    sql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   sql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
	}, logger)
	require.NoError(t, err)

	testOneMessage(t, publisher, subscriber)
}

func testOneMessage(t *testing.T, publisher message.Publisher, subscriber message.Subscriber) {
	topic := "test_" + watermill.NewULID()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, err := subscriber.Subscribe(ctx, topic)
	require.NoError(t, err)

	msg := message.NewMessage(watermill.NewULID(), []byte(`{"json": "field"}`))
	err = publisher.Publish(topic, msg)
	require.NoError(t, err)

	select {
	case received := <-messages:
		require.Equal(t, msg.UUID, received.UUID)
		require.Equal(t, msg.Payload, received.Payload)
		received.Ack()
	case <-time.After(time.Second * 5):
		t.Error("Didn't receive any messages")
	}
}

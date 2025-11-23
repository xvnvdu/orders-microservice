package kafka

import (
	"context"
	"errors"
	"testing"

	"orders/internal/mocks"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWriteMessage(t *testing.T) {
	t.Run("Successful write", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProducer := mocks.NewMockMessagesProducer(ctrl)

		ctx := context.Background()
		msg := []byte("Hello there, this is a test for Kafka writer !")
		expectedMsg := kafka.Message{
			Key:   nil,
			Value: msg,
		}

		mockProducer.EXPECT().
			WriteMessages(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, msgs ...kafka.Message) error {
				if len(msgs) != 1 {
					t.Errorf("Expected 1 message, got %d", len(msgs))
				}
				if string(msgs[0].Value) != string(expectedMsg.Value) {
					t.Errorf("Message content missmatch")
				}
				return nil
			}).
			Times(1)

		err := WriteMessage(mockProducer, ctx, msg)
		assert.NoError(t, err, "WriteMessage should not return an error on successful mock call")
	})

	t.Run("Failed write", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockProducer := mocks.NewMockMessagesProducer(ctrl)
		ctx := context.Background()
		msg := []byte("Another test for error check")
		expectedError := errors.New("Simulated Kafka error")

		mockProducer.EXPECT().
			WriteMessages(gomock.Any(), gomock.Any()).
			Return(expectedError).
			Times(1)

		err := WriteMessage(mockProducer, ctx, msg)
		assert.Error(t, err, "WriteMessage should return an error on failed mock call")
		assert.Equal(t, expectedError, err, "Returned error should match expected one")
	})
}

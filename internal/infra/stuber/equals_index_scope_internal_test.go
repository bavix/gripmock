package stuber

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIndexedCandidatesKeepStubsOfEveryServiceSpelling(t *testing.T) {
	t.Parallel()

	equalsStub := &Stub{
		ID:      uuid.New(),
		Service: "ChatService",
		Method:  "SendMessage",
		Input:   InputData{Equals: map[string]any{"text": "Hello", "user": "Alice"}},
		Output:  Output{Data: map[string]any{"reply": "hi"}},
	}

	streamStub := &Stub{
		ID:       uuid.New(),
		Service:  "chat.ChatService",
		Method:   "SendMessage",
		Priority: 98,
		Inputs:   []InputData{{Contains: map[string]any{"text": "ping"}}},
		Output:   Output{Data: map[string]any{"reply": "pong"}},
	}

	b := NewBudgerigar()
	b.PutMany(equalsStub, streamStub)

	result, err := b.FindByQuery(Query{
		Service: "chat.ChatService",
		Method:  "SendMessage",
		Input:   []map[string]any{{"text": "ping", "user": ""}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, streamStub.ID, result.Found().ID)
}

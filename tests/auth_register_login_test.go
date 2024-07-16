package tests

import (
	"testing"

	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/authpb"
	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/chatpb"
	"github.com/alexandernizov/grpcmessanger/tests/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HappyPath(t *testing.T) {
	const (
		name = "Alexander"
		pass = "Nizov"
	)
	ctx, st := suite.New(t)

	respRegister, err := st.AuthClient.Register(ctx, &authpb.RegisterReq{
		Name:     name,
		Password: pass},
	)
	require.NoError(t, err)
	assert.Equal(t, respRegister.GetRegistred(), true)

	respLogin, err := st.AuthClient.Login(ctx, &authpb.LoginReq{
		Name:     name,
		Password: pass},
	)
	require.NoError(t, err)
	sessionUuid, err := uuid.Parse(respLogin.SessionUuid)
	assert.NoError(t, err)
	assert.Equal(t, 16, len(sessionUuid))

	respNewChat, err := st.ChatClient.NewChat(ctx, &chatpb.NewChatReq{
		Readonly: chatpb.ReadOnly_TRUE,
		TtlSecs:  10,
	})
	require.NoError(t, err)
	chatUuid, err := uuid.Parse(respNewChat.Uuid)
	assert.NoError(t, err)
	assert.Equal(t, 16, len(chatUuid))

	respNewMessage, _ := st.ChatClient.NewMessage(ctx, &chatpb.NewMessageReq{
		ChatUuid: respNewChat.GetUuid(),
		Message:  "some message",
	})
	assert.True(t, respNewMessage.Published, "message not published")
}

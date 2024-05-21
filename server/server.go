package server

import (
	"context"
	"github.com/xissg/chat-room/proto/chat"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"sync"
)

type server struct {
	chat.UnimplementedChatServiceServer
	users    map[string]chat.ChatService_ChatServer
	match    map[string]string
	userLock sync.Mutex
}

func newServer() *server {
	return &server{
		users: make(map[string]chat.ChatService_ChatServer),
		match: make(map[string]string),
	}
}

// 注册用户,直接存储在内存中
func (s *server) RegisterUser(ctx context.Context, user *chat.User) (*chat.RegisterResponse, error) {
	s.userLock.Lock()
	defer s.userLock.Unlock()

	if _, ok := s.users[user.Username]; ok {
		return &chat.RegisterResponse{
			Success: false,
			Message: "User  already exists",
		}, nil
	}
	return &chat.RegisterResponse{Success: true, Message: "User registered success"}, nil
}

// 直接按用户名进行匹配,后续进行匹配优化
func (s *server) MatchUser(ctx context.Context, req *chat.MatchRequest) (*chat.MathResponse, error) {
	s.userLock.Lock()
	defer s.userLock.Unlock()

	//判断是否注册
	if _, ok := s.users[req.Username]; !ok {
		return &chat.MathResponse{
			Success: false,
			Message: "User not register yet",
		}, nil
	}
	if _, ok := s.match[req.TargetUser]; !ok {
		return &chat.MathResponse{
			Success: false,
			Message: "Target user not register yet",
		}, nil
	}

	s.match[req.Username] = req.TargetUser
	s.match[req.TargetUser] = req.Username
	return &chat.MathResponse{
		Success: true,
		Message: "Match success",
	}, nil
}

// 用户聊天
func (s *server) Chat(stream chat.ChatService_ChatServer) error {
	var username string

	for {
		msg, err := stream.Recv()
		//接收成功
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.userLock.Lock()
		if username == "" {
			username = msg.From
			s.users[username] = stream
		}

		targetStream, exists := s.users[msg.To]
		s.userLock.Unlock()

		if exists && targetStream != nil {
			targetStream.Send(msg)
		} else {
			stream.Send(&chat.ChatMessage{
				From:    msg.To,
				To:      msg.From,
				Message: "User not online",
			})
		}
	}
}

func start() {
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("error listening %v", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServiceServer(s, newServer())

	log.Printf("Server is running on port 8081")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

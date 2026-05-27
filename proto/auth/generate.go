package authpb

//go:generate mockgen -source=auth_grpc.pb.go -destination=mock/auth_client_mock.go -package=mock AuthServiceClient
